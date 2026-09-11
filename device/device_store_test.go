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
	info := &Device{
		DeviceLite: DeviceLite{
			Platform:   PlatformIOS,
			ClientKind: ClientKindMobile,
			DeviceLiveInfo: DeviceLiveInfo{
				RamAvailMB:  1024, // 不应入库
				NetworkType: NetworkTypeWifi,
				Lng:         116.4,
			},
		},
		App:      DeviceAppInfo{Code: "hoper", Version: "2.0.0"},
		Hardware: DeviceHardwareInfo{ModelName: "iPhone 16"},
		ID:       DeviceIDInfo{DID: "biz-did"},
		OS:       DeviceOSInfo{Name: "iOS", Version: "18.0"},
		Host:     DeviceHostInfo{RamMB: 8192},
		Network:  DeviceNetworkInfo{Carrier: "CMCC"},
	}
	// 主键与各域主键都由调用方预计算（生产实现是 protobuf 确定性 MD5）；
	// 本测试只关心存储行为，用固定串即可。
	const deviceID = "fixture-stable-md5"
	ids := DomainIDs{
		App:      "fixture-app-md5",
		Hardware: "fixture-hw-md5",
		Ident:    "fixture-ident-md5",
		OS:       "fixture-os-md5",
		Host:     "fixture-host-md5",
		Network:  "fixture-net-md5",
		// Web 留空：空域不应建行。
	}
	id1, err := Upsert(db, info, deviceID, ids)
	if err != nil {
		t.Fatal(err)
	}
	if id1 == "" {
		t.Fatal("empty id")
	}
	id2, err := Upsert(db, info, deviceID, ids)
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
	var webN int64
	if err := db.Model(&DeviceWebRow{}).Count(&webN).Error; err != nil {
		t.Fatal(err)
	}
	if webN != 0 {
		t.Fatalf("empty domain must not create rows, got %d", webN)
	}
}
