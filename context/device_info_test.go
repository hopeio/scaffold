/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 */

package context

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDeviceFromHeaderJSON(t *testing.T) {
	h := http.Header{}
	h.Set("Device-Info", `{
		"platform":"ios","clientKind":"mobile",
		"app":{"code":"hoper","version":"2.0.0"},
		"hardware":{"modelName":"iPhone 16"},
		"os":{"name":"iOS","version":"18.0","arch":"arm64"},
		"id":{
			"did":"biz-did","idfv":"idfv-1","idfa":"idfa-1",
			"aaid":"aaid-1","oaid":"oaid-1","imei":"860000000000001",
			"mac":"AA:BB:CC:DD:EE:FF"
		}
	}`)
	h.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	info := DeviceFromHeader(h)
	if info == nil {
		t.Fatal("nil")
	}
	if info.Platform != PlatformIOS || info.ClientKind != ClientKindMobile {
		t.Fatalf("platform: %+v", info)
	}
	if info.DisplayName() != "iPhone 16" {
		t.Fatalf("display: %q", info.DisplayName())
	}
	if info.OSDisplay() != "iOS 18.0" {
		t.Fatalf("os: %q", info.OSDisplay())
	}
	if info.PrimaryDeviceNo() != "biz-did" {
		t.Fatalf("primary: %q", info.PrimaryDeviceNo())
	}
	if info.ID.IDFV != "idfv-1" || info.ID.OAID != "oaid-1" || info.ID.AAID != "aaid-1" {
		t.Fatalf("ids: %+v", info.ID)
	}
	if info.ID.IMEI != "860000000000001" || info.ID.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("imei/mac: %+v", info.ID)
	}
	if info.Network.IP.String() != "10.0.0.1" {
		t.Fatalf("xff ip: %v", info.Network.IP)
	}
	lite := info.Lite()
	if lite.Platform != "ios" || lite.ClientKind != "mobile" || lite.AppCode != "hoper" {
		t.Fatalf("lite: %+v", lite)
	}
}

func TestDeviceFromHeaderEmpty(t *testing.T) {
	if DeviceFromHeader(http.Header{}) != nil {
		t.Fatal("expected nil")
	}
}

func TestAAIDGAIDAlias(t *testing.T) {
	info := &DeviceInfo{ID: DeviceIDInfo{AAID: "x"}}
	info.Normalize()
	if info.ID.GAID != "x" {
		t.Fatalf("gaid alias: %q", info.ID.GAID)
	}
}

func TestTriState(t *testing.T) {
	info := &DeviceInfo{
		Hardware: DeviceHardwareInfo{IsPhysical: TriFalse},
		ID:       DeviceIDInfo{IDFATracking: TriTrue},
	}
	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var out DeviceInfo
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Hardware.IsPhysical.IsFalse() || !out.ID.IDFATracking.IsTrue() || out.Hardware.IsLowRam.IsSet() {
		t.Fatalf("tristate: %+v", out)
	}
	if TriFromBool(true) != TriTrue || TriFromBool(false) != TriFalse {
		t.Fatal("TriFromBool")
	}
}

func TestHostLiveNested(t *testing.T) {
	raw := []byte(`{
		"platform":"macos",
		"host":{"ramMb":16384,"diskTotalBytes":500000000000,"live":{"ramAvailMb":4096,"diskFreeBytes":100}}
	}`)
	var info DeviceInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		t.Fatal(err)
	}
	if info.Host.RamMB != 16384 || info.Host.DiskTotalB != 500000000000 {
		t.Fatalf("static: %+v", info.Host)
	}
	if info.Host.Live.RamAvailMB != 4096 || info.Host.Live.DiskFreeB != 100 {
		t.Fatalf("live: %+v", info.Host.Live)
	}
}
