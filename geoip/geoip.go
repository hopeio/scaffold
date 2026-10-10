// Package geoip resolves a client IP to a coarse region (country / province /
// city) using ip2region xdb data. Pure logic, no global or DB dependency, so it
// stays unit-testable; the databases are injected by the caller per IP version.
//
// Two read modes are supported for each version (see Source): a whole xdb kept
// in memory (no IO per lookup, resident RSS equal to the file size) or an xdb
// file read on demand (a couple of small reads per lookup, negligible memory,
// optionally sped up by a preloaded vector index). Buffer mode is read-only and
// stateless, and file mode opens its own handle per lookup because the
// underlying xdb.Searcher is not thread safe.
package geoip

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// Region is the split view of an ip2region result. The xdb encodes unknown
// fields as the literal "0"; those are normalised to empty here.
type Region struct {
	Country  string // e.g. 中国 / United States
	Province string // e.g. 广东省 / California
	City     string // e.g. 深圳市; often "0" for overseas
	ISP      string // e.g. 阿里云; informational only
	ISO      string // 2-letter country code, e.g. CN / US
}

// IsMainland reports whether the IP resolves to mainland China, the only range
// that carries a national administrative code we can map to.
func (r Region) IsMainland() bool {
	return r.ISO == "" || strings.EqualFold(r.ISO, "CN")
}

// Source tells a Searcher where one IP version's xdb comes from and how to read
// it. Exactly one of Content / File is honoured; Content wins when both are set.
// A zero Source disables lookups for that version.
type Source struct {
	// Content is an xdb loaded into memory (see LoadFile). Fastest, but the
	// bytes stay resident for the process lifetime (~11 MB for IPv4, ~36 MB for
	// IPv6 in ip2region v3).
	Content []byte

	// File is an xdb path opened and closed per lookup. Keeps the database out
	// of RSS at the cost of a small read per search (benchmarked ~10 us against
	// ~0.2 us for Content on Apple M4), which suits low-rate IP attribution.
	File string

	// VectorIndex preloads the xdb vector index for File mode: a fixed-size
	// index block stays resident (much smaller than the database) so each search
	// needs one fewer IO. Ignored in Content mode.
	VectorIndex bool
}

func (src Source) enabled() bool { return src.Content != nil || src.File != "" }

// db is the per-version lookup state, built once by New.
type db struct {
	ipv     *xdb.Version
	content []byte
	file    string
	vIndex  []byte
	on      bool
}

// Searcher resolves IPv4 and IPv6 from ip2region xdb data. It is safe to share
// across goroutines: buffer-mode lookups hold no state, and file-mode lookups
// open a dedicated handle per search.
type Searcher struct {
	ipv4 db
	ipv6 db
}

// ErrNoDatabase means no xdb was loaded for the version being resolved.
var ErrNoDatabase = errors.New("geoip: no xdb loaded")

// ErrVersionMismatch means the xdb fed to a slot is of a different IP version
// than that slot (e.g. the IPv4 file configured as the IPv6 source). It is
// returned by New, so the mistake surfaces at startup rather than as garbage or
// silent misses on the write path.
var ErrVersionMismatch = errors.New("geoip: xdb ip version mismatch")

// New builds a Searcher from per-version sources. File sources are validated up
// front (openable, right IP version, vector index loadable); Content sources
// only get their header version checked, since they are already in memory.
func New(ipv4, ipv6 Source) (*Searcher, error) {
	s := &Searcher{}
	if err := s.ipv4.init(xdb.IPv4, ipv4); err != nil {
		return nil, err
	}
	if err := s.ipv6.init(xdb.IPv6, ipv6); err != nil {
		return nil, err
	}
	return s, nil
}

func (d *db) init(ipv *xdb.Version, src Source) error {
	d.ipv = ipv
	if !src.enabled() {
		return nil
	}
	if src.Content != nil {
		head, err := xdb.LoadHeaderFromBuff(src.Content)
		if err != nil {
			return fmt.Errorf("geoip: read xdb header: %w", err)
		}
		if head.IPVersion != ipv.Id {
			return fmt.Errorf("geoip: xdb ip version %d, want %d: %w", head.IPVersion, ipv.Id, ErrVersionMismatch)
		}
		d.content, d.on = src.Content, true
		return nil
	}

	handle, err := os.Open(src.File)
	if err != nil {
		return fmt.Errorf("geoip: open xdb %s: %w", src.File, err)
	}
	head, err := xdb.LoadHeader(handle)
	handle.Close()
	if err != nil {
		return fmt.Errorf("geoip: read xdb header %s: %w", src.File, err)
	}
	if head.IPVersion != ipv.Id {
		return fmt.Errorf("geoip: %s is ip version %d, want %d: %w", src.File, head.IPVersion, ipv.Id, ErrVersionMismatch)
	}
	if src.VectorIndex {
		vIndex, err := xdb.LoadVectorIndexFromFile(src.File)
		if err != nil {
			return fmt.Errorf("geoip: load vector index %s: %w", src.File, err)
		}
		d.vIndex = vIndex
	}
	d.file, d.on = src.File, true
	return nil
}

// LoadFile reads an xdb file fully into memory and returns its bytes.
func LoadFile(path string) ([]byte, error) {
	return xdb.LoadContentFromFile(path)
}

// VectorIndexBytes returns the size of the preloaded vector index block, so
// callers can report how much a File+VectorIndex source keeps resident.
func VectorIndexBytes() int {
	return xdb.VectorIndexRows * xdb.VectorIndexCols * xdb.VectorIndexSize
}

// Search resolves an IP string. It returns ErrNoDatabase when no xdb matches the
// IP version, and a zero Region (ok=false) for a valid IP that hits an empty
// segment. An unparseable IP returns an error.
func (s *Searcher) Search(ip string) (Region, bool, error) {
	d := s.pick(ip)
	if d == nil {
		return Region{}, false, ErrNoDatabase
	}
	raw, err := d.search(ip)
	if err != nil {
		return Region{}, false, err
	}
	if raw == "" {
		return Region{}, false, nil
	}
	return parse(raw), true, nil
}

// search runs one lookup. Each call builds its own xdb.Searcher: in buffer mode
// that is allocation-only and opens nothing, in file mode it owns a handle that
// is closed before returning. Either way no state is shared between goroutines.
func (d *db) search(ip string) (string, error) {
	if d.content != nil {
		searcher, err := xdb.NewWithBuffer(d.ipv, d.content)
		if err != nil {
			return "", err
		}
		return searcher.Search(ip)
	}
	searcher, err := xdb.NewSearcher(d.ipv, d.file, d.vIndex, nil)
	if err != nil {
		return "", err
	}
	defer searcher.Close()
	return searcher.Search(ip)
}

// pick selects the xdb for the IP's version, or nil when that version has no
// database. An unparseable IP is treated as IPv4 and then rejected by the
// search itself.
func (s *Searcher) pick(ip string) *db {
	ipv, err := xdb.VersionFromIP(ip)
	if err == nil && ipv != nil && ipv.Id == xdb.IPv6VersionNo {
		if !s.ipv6.on {
			return nil
		}
		return &s.ipv6
	}
	if !s.ipv4.on {
		return nil
	}
	return &s.ipv4
}

// parse splits the `country|province|city|isp|ISO` layout emitted by ip2region
// v3, mapping the sentinel "0" to an empty field.
func parse(raw string) Region {
	f := strings.Split(raw, "|")
	// Region layout is fixed at 5 fields; tolerate a shorter tail defensively.
	get := func(i int) string {
		if i >= len(f) {
			return ""
		}
		if v := f[i]; v != "0" {
			return v
		}
		return ""
	}
	r := Region{
		Country:  get(0),
		Province: get(1),
		City:     get(2),
		ISP:      get(3),
		ISO:      get(4),
	}
	return r
}
