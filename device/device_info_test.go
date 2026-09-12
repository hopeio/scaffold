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
	h.Set(HeaderDeviceDynamicInfo, "wifi")

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
	if lite.NetworkType != NetworkTypeWifi {
		t.Fatalf("netType: %q", lite.NetworkType)
	}
}

func TestLiteFromHeaderEmpty(t *testing.T) {
	if !LiteFromHeader(http.Header{}).Empty() {
		t.Fatal("empty headers")
	}
	if LiteFromHeader(http.Header{}).NetworkType != NetworkTypeUnspecified {
		t.Fatal("empty netType")
	}
}

func TestNetworkLiveFromHeaderReplace(t *testing.T) {
	info := &UserDevice{
		DeviceLite: DeviceLite{
			Platform:       PlatformIOS,
			DeviceLiveInfo: DeviceLiveInfo{Area: "old", Lng: 1, Lat: 2, NetworkType: NetworkTypeUnknown},
		},
	}
	h := make(http.Header)
	h.Set("Location", ";;new")
	h.Set(HeaderDeviceDynamicInfo, "wifi;256;99")
	lite := LiteFromHeader(h)
	info.DeviceLiveInfo.Lng, info.DeviceLiveInfo.Lat, info.DeviceLiveInfo.Area = lite.Lng, lite.Lat, lite.Area
	info.DeviceLiveInfo.NetworkType = lite.NetworkType
	info.DeviceLiveInfo.RamAvailMB = lite.RamAvailMB
	info.DeviceLiveInfo.DiskFreeB = lite.DiskFreeB
	if lite.UserAgent != "" {
		info.Web.UserAgent = lite.UserAgent
	}
	if info.Platform != PlatformIOS {
		t.Fatal("stable fields must stay")
	}
	if info.DeviceLiveInfo.Area != "new" {
		t.Fatalf("area overwrite: %q", info.DeviceLiveInfo.Area)
	}
	if info.DeviceLiveInfo.Lng != 0 || info.DeviceLiveInfo.Lat != 0 {
		t.Fatalf("stale loc kept: %v %v", info.DeviceLiveInfo.Lng, info.DeviceLiveInfo.Lat)
	}
	if info.DeviceLiveInfo.NetworkType != NetworkTypeWifi {
		t.Fatalf("net type should update: %q", info.DeviceLiveInfo.NetworkType)
	}
	if info.DeviceLiveInfo.RamAvailMB != 256 || info.DeviceLiveInfo.DiskFreeB != 99 {
		t.Fatalf("live: %+v", info.DeviceLiveInfo)
	}
}

func TestEmptyIgnoresLive(t *testing.T) {
	d := &UserDevice{Web: DeviceWebInfo{UserAgent: "ua"}}
	if !d.Empty() {
		t.Fatal("UA-only is not a stable identity")
	}
	if !d.HasLive() {
		t.Fatal("UA is live")
	}
}
