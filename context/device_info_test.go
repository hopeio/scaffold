/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 */

package context

import (
	"net/http"
	"testing"
)

func TestDeviceLegacyCSV(t *testing.T) {
	info := Device("Pixel 8,Android 14", "hoper,1.2.3", "上海", "121.5,31.2", "ua", "1.2.3.4")
	if info == nil {
		t.Fatal("nil")
	}
	if info.Device != "Pixel 8" || info.OS != "Android 14" {
		t.Fatalf("device/os: %+v", info)
	}
	if info.AppCode != "hoper" || info.AppVer != "1.2.3" {
		t.Fatalf("app: %+v", info)
	}
	if info.Lng != 121.5 || info.Lat != 31.2 {
		t.Fatalf("loc: %+v", info)
	}
	if info.IP.String() != "1.2.3.4" {
		t.Fatalf("ip: %v", info.IP)
	}
}

func TestDeviceFromHeaderJSON(t *testing.T) {
	h := http.Header{}
	h.Set("Device-Info", `{"platform":"ios","clientKind":"mobile","modelName":"iPhone 16","osName":"iOS","osVersion":"18.0","appCode":"hoper","appVer":"2.0.0","deviceId":"idfv-1","arch":"arm64"}`)
	h.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	info := DeviceFromHeader(h)
	if info == nil {
		t.Fatal("nil")
	}
	if info.Platform != PlatformIOS || info.ClientKind != ClientKindMobile {
		t.Fatalf("platform: %+v", info)
	}
	if info.Device != "iPhone 16" {
		t.Fatalf("normalized device: %q", info.Device)
	}
	if info.OS != "iOS 18.0" {
		t.Fatalf("normalized os: %q", info.OS)
	}
	if info.DeviceID != "idfv-1" || info.Arch != "arm64" {
		t.Fatalf("ext fields: %+v", info)
	}
	if info.IP.String() != "10.0.0.1" {
		t.Fatalf("xff ip: %v", info.IP)
	}
}
