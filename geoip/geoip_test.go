package geoip

import (
	"os"
	"path/filepath"
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
	s := New(nil, nil)
	if _, ok, err := s.Search("1.2.3.4"); ok || err != ErrNoDatabase {
		t.Fatalf("want ErrNoDatabase, got ok=%v err=%v", ok, err)
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
	s := New(buff, nil)
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
		t.Errorf("ipv6 with no v6 buffer: want ErrNoDatabase, got %v", err)
	}
}
