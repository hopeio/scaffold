package geoip

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// xdbPath resolves the IPv4 xdb from env or the ip2region module cache. It
// returns "" when the file is absent so callers can skip live-search tests.
func xdbPath() string {
	if p := os.Getenv("IP2REGION_XDB"); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, "go/pkg/mod/github.com/lionsoul2014/ip2region@v3.18.0+incompatible/data/ip2region_v4.xdb")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// xdbV6Path resolves the IPv6 xdb the same way; the file is much larger than the
// IPv4 one so it is not always vendored locally.
func xdbV6Path() string {
	if p := os.Getenv("IP2REGION_XDB_V6"); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, "go/pkg/mod/github.com/lionsoul2014/ip2region@v3.18.0+incompatible/data/ip2region_v6.xdb")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// BenchmarkSearch quantifies the trade the two read modes make for one another:
// the memory copy resolves with no IO, the file handle pays a few small reads.
func BenchmarkSearch(b *testing.B) {
	path := xdbPath()
	if path == "" {
		b.Skip("ip2region v4 xdb not available")
	}
	buff, err := LoadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	for _, mode := range []struct {
		name string
		src  Source
	}{
		{"memory", Source{Content: buff}},
		{"file", Source{File: path, VectorIndex: true}},
	} {
		s, err := New(mode.src, Source{})
		if err != nil {
			b.Fatal(err)
		}
		b.Run(mode.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, _, err := s.Search("114.114.114.114"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		raw  string
		want Region
	}{
		{"中国|江苏省|南京市|0|CN", Region{Country: "中国", Province: "江苏省", City: "南京市", ISO: "CN"}},
		{"中国|北京市|北京市|0|CN", Region{Country: "中国", Province: "北京市", City: "北京市", ISO: "CN"}},
		{"United States|California|0|Google LLC|US", Region{Country: "United States", Province: "California", ISP: "Google LLC", ISO: "US"}},
		{"中国|广东省|深圳市|联通|CN", Region{Country: "中国", Province: "广东省", City: "深圳市", ISP: "联通", ISO: "CN"}},
	}
	for _, c := range cases {
		if got := parse(c.raw); got != c.want {
			t.Errorf("parse(%q) = %+v, want %+v", c.raw, got, c.want)
		}
	}
}

func TestIsMainland(t *testing.T) {
	if !(Region{ISO: "CN"}).IsMainland() || !(Region{}).IsMainland() {
		t.Error("CN/empty should be mainland")
	}
	if (Region{ISO: "US"}).IsMainland() {
		t.Error("US should not be mainland")
	}
}

func TestSearcherNoDatabase(t *testing.T) {
	s, err := New(Source{}, Source{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.Search("1.2.3.4"); ok || err != ErrNoDatabase {
		t.Fatalf("want ErrNoDatabase, got ok=%v err=%v", ok, err)
	}
	if _, _, err := s.Search("2400:3200::1"); err != ErrNoDatabase {
		t.Fatalf("ipv6 with no database: want ErrNoDatabase, got %v", err)
	}
}

func TestNewRejectsWrongIPVersion(t *testing.T) {
	path := xdbPath()
	if path == "" {
		t.Skip("ip2region v4 xdb not available")
	}
	buff, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The IPv4 database must not be accepted into the IPv6 slot.
	if _, err := New(Source{Content: buff}, Source{Content: buff}); err == nil ||
		!errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("want ErrVersionMismatch, got %v", err)
	}
	if _, err := New(Source{File: path}, Source{File: path + ".missing"}); err == nil {
		t.Fatal("missing file: want error, got nil")
	}
}

func TestSearch(t *testing.T) {
	path := xdbPath()
	if path == "" {
		t.Skip("ip2region v4 xdb not available")
	}
	buff, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(Source{Content: buff}, Source{})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		ip       string
		province string
		city     string
		mainland bool
	}{
		{"114.114.114.114", "江苏省", "南京市", true},
		{"1.2.4.8", "北京市", "北京市", true},
		{"8.8.8.8", "California", "", false},
	}
	for _, c := range cases {
		r, ok, err := s.Search(c.ip)
		if err != nil || !ok {
			t.Fatalf("search %s: ok=%v err=%v", c.ip, ok, err)
		}
		if r.Province != c.province || r.City != c.city {
			t.Errorf("search %s = %+v, want province=%q city=%q", c.ip, r, c.province, c.city)
		}
		if r.IsMainland() != c.mainland {
			t.Errorf("search %s IsMainland=%v want %v", c.ip, r.IsMainland(), c.mainland)
		}
	}
	// IPv6 must not resolve against an IPv4-only searcher.
	if _, _, err := s.Search("2400:3200::1"); err != ErrNoDatabase {
		t.Errorf("ipv6 with no v6 database: want ErrNoDatabase, got %v", err)
	}
}

// TestSearchFromFile checks the file-handle mode (no whole database resident) with a
// preloaded vector index against the buffer results.
func TestSearchFromFile(t *testing.T) {
	path := xdbPath()
	if path == "" {
		t.Skip("ip2region v4 xdb not available")
	}
	s, err := New(Source{File: path, VectorIndex: true}, Source{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ ip, province, city string }{
		{"114.114.114.114", "江苏省", "南京市"},
		{"1.2.4.8", "北京市", "北京市"},
		{"8.8.8.8", "California", ""},
	} {
		r, ok, err := s.Search(c.ip)
		if err != nil || !ok {
			t.Fatalf("search %s: ok=%v err=%v", c.ip, ok, err)
		}
		if r.Province != c.province || r.City != c.city {
			t.Errorf("file mode %s = %+v, want province=%q city=%q", c.ip, r, c.province, c.city)
		}
	}
}

// TestMixedSources mirrors the production wiring: the IPv4 xdb resident in memory
// and the IPv6 xdb read from file with a preloaded vector index. Searches must
// route per version and stay correct when one Searcher is shared across goroutines.
func TestMixedSources(t *testing.T) {
	v4Path, v6Path := xdbPath(), xdbV6Path()
	if v4Path == "" || v6Path == "" {
		t.Skip("ip2region v4/v6 xdb not both available")
	}
	buff, err := LoadFile(v4Path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(Source{Content: buff}, Source{File: v6Path, VectorIndex: true})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		ip, province string
		city         string
	}{
		{"114.114.114.114", "江苏省", "南京市"},    // IPv4 from the memory copy
		{"2400:3200::1", "浙江省", "杭州市"},       // IPv6 from the file
		{"::ffff:8.8.8.8", "California", ""}, // 4-in-6 must use the IPv4 xdb
	}

	const goroutines = 8
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 20; n++ {
				for _, c := range cases {
					r, ok, err := s.Search(c.ip)
					if err != nil || !ok {
						t.Errorf("search %s: ok=%v err=%v", c.ip, ok, err)
						return
					}
					if r.Province != c.province || r.City != c.city {
						t.Errorf("search %s = %+v, want province=%q city=%q",
							c.ip, r, c.province, c.city)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

// TestSearchIPv6 checks the opposite half of the routing: with only an IPv6
// database loaded, an IPv4 address is refused instead of being guessed.
func TestSearchIPv6(t *testing.T) {
	path := xdbV6Path()
	if path == "" {
		t.Skip("ip2region v6 xdb not available")
	}
	s, err := New(Source{}, Source{File: path, VectorIndex: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Search("1.2.3.4"); err != ErrNoDatabase {
		t.Errorf("ipv4 with no v4 database: want ErrNoDatabase, got %v", err)
	}
	for _, c := range []struct{ ip, province, city string }{
		{"2400:3200::1", "浙江省", "杭州市"}, // Ali DNS IPv6
		{"240e:978::1", "北京市", "北京市"},  // China Telecom IPv6
	} {
		r, ok, err := s.Search(c.ip)
		if err != nil {
			t.Fatalf("search %s: %v", c.ip, err)
		}
		if !ok || r.Province != c.province || r.City != c.city {
			t.Errorf("search %s = %+v (ok=%v), want province=%q city=%q",
				c.ip, r, ok, c.province, c.city)
		}
		if !r.IsMainland() {
			t.Errorf("search %s: want mainland, got %+v", c.ip, r)
		}
	}
}
