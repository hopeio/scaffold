// Package geoip resolves a client IP to a coarse region (country / province /
// city) using an ip2region xdb kept fully in memory. Pure logic, no global or
// DB dependency, so it stays unit-testable; the xdb path is injected by the
// caller. Only the buffer mode is used, which makes a Searcher read-only and
// safe to share across goroutines.
package geoip

import (
	"errors"
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

// Searcher resolves IPs from an in-memory xdb buffer.
type Searcher struct {
	ipv4 []byte
	ipv6 []byte
}

// ErrNoDatabase means no xdb buffer was loaded, so nothing can be resolved.
var ErrNoDatabase = errors.New("geoip: no xdb loaded")

// New builds a Searcher from already-loaded xdb buffers. A nil buffer for a
// version disables lookups for that version; callers normally load only IPv4.
func New(ipv4, ipv6 []byte) *Searcher {
	return &Searcher{ipv4: ipv4, ipv6: ipv6}
}

// LoadFile reads an xdb file fully into memory and returns its bytes.
func LoadFile(path string) ([]byte, error) {
	return xdb.LoadContentFromFile(path)
}

// Search resolves an IP string. It returns ErrNoDatabase when no buffer
// matches the IP version, and a zero Region (ok=false) for a valid IP that hits
// an empty segment. An unparseable IP returns an error.
func (s *Searcher) Search(ip string) (Region, bool, error) {
	ipv, buff := pick(s, ip)
	if buff == nil {
		return Region{}, false, ErrNoDatabase
	}
	// Buffer-mode Searcher holds no per-call state and opens no handle, so it
	// is created cheaply per lookup and safe under concurrency.
	searcher, err := xdb.NewWithBuffer(ipv, buff)
	if err != nil {
		return Region{}, false, err
	}
	raw, err := searcher.Search(ip)
	if err != nil {
		return Region{}, false, err
	}
	if raw == "" {
		return Region{}, false, nil
	}
	return parse(raw), true, nil
}

func pick(s *Searcher, ip string) (*xdb.Version, []byte) {
	ipv, err := xdb.VersionFromIP(ip)
	if err != nil || ipv == nil {
		return xdb.IPv4, s.ipv4
	}
	if ipv.Id == xdb.IPv6VersionNo {
		return xdb.IPv6, s.ipv6
	}
	return xdb.IPv4, s.ipv4
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
