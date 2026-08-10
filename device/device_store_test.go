/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 */

package device

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpsertSplitTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_upsert?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrateDeviceTables(db); err != nil {
		t.Fatal(err)
	}
	info := &DeviceInfo{
		Platform:   PlatformIOS,
		ClientKind: ClientKindMobile,
		App:        DeviceAppInfo{Code: "hoper", Version: "2.0.0"},
		Hardware:   DeviceHardwareInfo{ModelName: "iPhone 16"},
		ID:         DeviceIDInfo{DID: "biz-did"},
		OS:         DeviceOSInfo{Name: "iOS", Version: "18.0"},
		Host:       DeviceHostInfo{RamMB: 8192},
		HostLive:   DeviceHostLiveInfo{RamAvailMB: 1024}, // 不应入库
		Network:    DeviceNetworkInfo{Carrier: "CMCC"},
		NetworkLive: DeviceNetworkLiveInfo{
			NetworkType: "wifi",
			Lng:         116.4,
		},
	}
	id1, err := Upsert(db, info)
	if err != nil {
		t.Fatal(err)
	}
	if id1 == "" {
		t.Fatal("empty id")
	}
	id2, err := Upsert(db, info)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("idempotent md5: %s vs %s", id1, id2)
	}
	var n int64
	if err := db.Model(&DeviceRow{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("device rows: %d", n)
	}
	var appN int64
	if err := db.Model(&DeviceAppRow{}).Count(&appN).Error; err != nil {
		t.Fatal(err)
	}
	if appN != 1 {
		t.Fatalf("app rows: %d", appN)
	}
}
