/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 */

package context

import (
	"net/http"
	"testing"
)

func TestDeviceFromHeaderJSON(t *testing.T) {
	h := http.Header{}
	h.Set("Device-Info", `{
		"platform":"ios","clientKind":"mobile",
		"modelName":"iPhone 16","osName":"iOS","osVersion":"18.0",
		"appCode":"hoper","appVer":"2.0.0",
		"idfv":"idfv-1","idfa":"idfa-1","did":"biz-did",
		"aaid":"aaid-1","oaid":"oaid-1","imei":"860000000000001",
		"mac":"AA:BB:CC:DD:EE:FF","arch":"arm64"
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
	if info.IDFV != "idfv-1" || info.IDFA != "idfa-1" || info.OAID != "oaid-1" || info.AAID != "aaid-1" {
		t.Fatalf("ids: %+v", info)
	}
	if info.IMEI != "860000000000001" || info.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("imei/mac: %+v", info)
	}
	if info.IP.String() != "10.0.0.1" {
		t.Fatalf("xff ip: %v", info.IP)
	}
}

func TestDeviceFromHeaderEmpty(t *testing.T) {
	if DeviceFromHeader(http.Header{}) != nil {
		t.Fatal("expected nil")
	}
}

func TestAAIDGAIDAlias(t *testing.T) {
	info := &DeviceInfo{AAID: "x"}
	info.Normalize()
	if info.GAID != "x" {
		t.Fatalf("gaid alias: %q", info.GAID)
	}
}
