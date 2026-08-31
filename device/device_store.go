/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 */

package device

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 分表行：各域内容寻址，主键为内容 MD5（32 hex）。
// HostLive / NetworkLive 不入库。

type DeviceAppRow struct {
	ID string `json:"id" gorm:"primaryKey;size:32"`
	DeviceAppInfo
}

func (DeviceAppRow) TableName() string { return "device_app" }

type DeviceHardwareRow struct {
	ID string `json:"id" gorm:"primaryKey;size:32"`
	DeviceHardwareInfo
}

func (DeviceHardwareRow) TableName() string { return "device_hardware" }

type DeviceIdentRow struct {
	ID string `json:"id" gorm:"primaryKey;size:32"`
	DeviceIDInfo
}

func (DeviceIdentRow) TableName() string { return "device_ident" }

type DeviceOSRow struct {
	ID string `json:"id" gorm:"primaryKey;size:32"`
	DeviceOSInfo
}

func (DeviceOSRow) TableName() string { return "device_os" }

type DeviceHostRow struct {
	ID string `json:"id" gorm:"primaryKey;size:32"`
	DeviceHostInfo
}

func (DeviceHostRow) TableName() string { return "device_host" }

type DeviceNetworkRow struct {
	ID string `json:"id" gorm:"primaryKey;size:32"`
	DeviceNetworkInfo
}

func (DeviceNetworkRow) TableName() string { return "device_network" }

type DeviceWebRow struct {
	ID string `json:"id" gorm:"primaryKey;size:32"`
	DeviceWebInfo
}

func (DeviceWebRow) TableName() string { return "device_web" }

// DeviceRow 设备主表：引用各域 MD5，不含 HostLive / NetworkLive。
type DeviceRow struct {
	ID         string            `json:"id" gorm:"primaryKey;size:32"`
	Platform   string            `json:"platform" gorm:"size:32"`
	ClientKind string            `json:"clientKind" gorm:"size:32"`
	AppID      string            `json:"appId" gorm:"size:32;index"`
	HardwareID string            `json:"hardwareId" gorm:"size:32;index"`
	IdentID    string            `json:"identId" gorm:"size:32;index"`
	OsID       string            `json:"osId" gorm:"size:32;index"`
	HostID     string            `json:"hostId" gorm:"size:32;index"`
	NetworkID  string            `json:"networkId" gorm:"size:32;index"`
	WebID      string            `json:"webId" gorm:"size:32;index"`
	Ext        map[string]string `json:"ext,omitempty" gorm:"serializer:json"`
}

func (DeviceRow) TableName() string { return "device" }

// ContentMD5 对可 JSON 序列化内容做 MD5（内容寻址主键）。
func ContentMD5(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:]), nil
}

func isZero(v any) bool {
	return reflect.ValueOf(v).IsZero()
}

func upsertByMD5[T any](db *gorm.DB, payload T) (id string, err error) {
	if isZero(payload) {
		return "", nil
	}
	id, err = ContentMD5(payload)
	if err != nil {
		return "", err
	}
	var row any
	switch p := any(payload).(type) {
	case DeviceAppInfo:
		row = &DeviceAppRow{ID: id, DeviceAppInfo: p}
	case DeviceHardwareInfo:
		row = &DeviceHardwareRow{ID: id, DeviceHardwareInfo: p}
	case DeviceIDInfo:
		row = &DeviceIdentRow{ID: id, DeviceIDInfo: p}
	case DeviceOSInfo:
		row = &DeviceOSRow{ID: id, DeviceOSInfo: p}
	case DeviceHostInfo:
		row = &DeviceHostRow{ID: id, DeviceHostInfo: p}
	case DeviceNetworkInfo:
		row = &DeviceNetworkRow{ID: id, DeviceNetworkInfo: p}
	case DeviceWebInfo:
		row = &DeviceWebRow{ID: id, DeviceWebInfo: p}
	default:
		return "", gorm.ErrInvalidData
	}
	err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error
	return id, err
}

// Upsert 分表写入稳定域；跳过 HostLive / NetworkLive。返回主表 MD5。
// id 为预计算的主表内容寻址主键（调用方需对 protobuf 稳定域二进制算好 MD5，
// 保证与服务端/客户端算法一致），必填；为空视为非法入参。
func Upsert(db *gorm.DB, info *DeviceInfo, id string) (string, error) {
	if info == nil || db == nil || id == "" {
		return "", gorm.ErrInvalidData
	}
	info.Normalize()

	appID, err := upsertByMD5(db, info.App)
	if err != nil {
		return "", err
	}
	hwID, err := upsertByMD5(db, info.Hardware)
	if err != nil {
		return "", err
	}
	identID, err := upsertByMD5(db, info.ID)
	if err != nil {
		return "", err
	}
	osID, err := upsertByMD5(db, info.OS)
	if err != nil {
		return "", err
	}
	hostID, err := upsertByMD5(db, info.Host)
	if err != nil {
		return "", err
	}
	netID, err := upsertByMD5(db, info.Network)
	if err != nil {
		return "", err
	}
	webID, err := upsertByMD5(db, info.Web)
	if err != nil {
		return "", err
	}

	row := DeviceRow{
		Platform:   info.Platform,
		ClientKind: info.ClientKind,
		AppID:      appID,
		HardwareID: hwID,
		IdentID:    identID,
		OsID:       osID,
		HostID:     hostID,
		NetworkID:  netID,
		WebID:      webID,
		Ext:        info.Ext,
	}
	row.ID = id
	err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
	return id, err
}

// AutoMigrateDeviceTables 迁移设备分表（不含 live）。
func AutoMigrateDeviceTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&DeviceAppRow{},
		&DeviceHardwareRow{},
		&DeviceIdentRow{},
		&DeviceOSRow{},
		&DeviceHostRow{},
		&DeviceNetworkRow{},
		&DeviceWebRow{},
		&DeviceRow{},
	)
}
