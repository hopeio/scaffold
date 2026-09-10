/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package device

import (
	"net/http"
	"testing"
)

func TestLiteFromHeader(t *testing.T) {
	h := make(http.Header)
	h.Set("Device-Info", `{"platform":"ios","id":{"did":"should-ignore"}}`)
	h.Set("Platform-Info", "ios;mobile;17.0")
	h.Set("App-Info", "rfv;1.2.3")
	h.Set("Location", "121.4;31.2;上海")
	h.Set("User-Agent", "app/1.0")
	h.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	h.Set("Device-Info-Md5", "abc")
	h.Set(HeaderRamAvail, "1024")
	h.Set(HeaderDiskFree, "500000")
	h.Set(HeaderNetworkType, "wifi")

	lite := LiteFromHeader(h)
	if lite.Empty() {
		t.Fatal("expected lite")
	}
	if lite.Platform != DevicePlatformIos || lite.ClientKind != ClientKindMobile || lite.Version != "17.0" || lite.AppCode != "rfv" || lite.AppVersion != "1.2.3" {
		t.Fatalf("app: %+v", lite)
	}
	if lite.Area != "上海" || lite.Lng != 121.4 || lite.Lat != 31.2 {
		t.Fatalf("loc: %+v", lite)
	}
	if lite.UserAgent != "app/1.0" {
		t.Fatalf("ua: %q", lite.UserAgent)
	}
	if lite.IP == nil || lite.IP.String() != "10.0.0.1" {
		t.Fatalf("ip: %v", lite.IP)
	}
	if lite.Md5 != "abc" {
		t.Fatalf("md5: %q", lite.Md5)
	}

	host, netType := HostLiveFromHeader(h)
	if !LivePresent(host, netType) {
		t.Fatal("expected host live")
	}
	if host.RamAvailMB != 1024 || host.DiskFreeB != 500000 || netType != "wifi" {
		t.Fatalf("host: %+v %q", host, netType)
	}
}

func TestLiteFromHeaderEmpty(t *testing.T) {
	if !LiteFromHeader(http.Header{}).Empty() {
		t.Fatal("empty headers")
	}
	host, netType := HostLiveFromHeader(http.Header{})
	if LivePresent(host, netType) {
		t.Fatal("empty host live")
	}
}

func TestHostLiveFromHeaderReplace(t *testing.T) {
	info := &Device{
		Platform:    PlatformIOS,
		NetworkLive: DeviceNetworkLiveInfo{Area: "old", Lng: 1, Lat: 2, NetworkType: "old"},
		HostLive:    DeviceHostLiveInfo{RamAvailMB: 8, DiskFreeB: 99},
	}
	h := make(http.Header)
	h.Set("Location", ";;new")
	h.Set(HeaderRamAvail, "256")
	lite := LiteFromHeader(h)
	host, netType := HostLiveFromHeader(h)
	info.HostLive = host
	info.NetworkLive.Lng, info.NetworkLive.Lat, info.NetworkLive.Area = lite.Lng, lite.Lat, lite.Area
	info.NetworkLive.NetworkType = netType
	if lite.UserAgent != "" {
		info.Web.UserAgent = lite.UserAgent
	}
	if info.Platform != PlatformIOS {
		t.Fatal("stable fields must stay")
	}
	if info.NetworkLive.Area != "new" {
		t.Fatalf("area overwrite: %q", info.NetworkLive.Area)
	}
	if info.NetworkLive.Lng != 0 || info.NetworkLive.Lat != 0 {
		t.Fatalf("stale loc kept: %v %v", info.NetworkLive.Lng, info.NetworkLive.Lat)
	}
	if info.HostLive.RamAvailMB != 256 || info.HostLive.DiskFreeB != 0 {
		t.Fatalf("hostLive: %+v", info.HostLive)
	}
	if info.NetworkLive.NetworkType != "" {
		t.Fatalf("stale net type: %q", info.NetworkLive.NetworkType)
	}
}

func TestEmptyIgnoresLive(t *testing.T) {
	d := &Device{Web: DeviceWebInfo{UserAgent: "ua"}}
	if !d.Empty() {
		t.Fatal("UA-only is not a stable identity")
	}
	if !d.HasLive() {
		t.Fatal("UA is live")
	}
}
