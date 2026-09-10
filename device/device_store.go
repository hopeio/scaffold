/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License. See the LICENSE file in the project root for more information.
 * @Created by jyb
 */

package device

import (
	"reflect"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 分表行：各域内容寻址，主键为内容哈希（32 hex）。
// HostLive / NetworkLive 不入库。
//
// 主键算法由调用方提供（见 Upsert 的 id / DomainIDs）：调用方对 protobuf
// 稳定域 / 各域二进制做确定性 MD5，保证与客户端逐字节一致。本包不引入
// 第二套（JSON）哈希，避免同名字段在两处口径不同。

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

// DeviceRow 设备主表：引用各域主键，不含 HostLive / NetworkLive。
// ID 即客户端 Device-Info-Md5（稳定域 protobuf MD5）。
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
	LastSeenAt time.Time         `json:"lastSeenAt" gorm:"index"`
}

func (DeviceRow) TableName() string { return "device" }

// DomainIDs 各域内容寻址主键（32 hex）；空串表示该域为零值、不建行。
// 调用方必须用与主表 id 相同的算法（protobuf 确定性序列化）对**各域子消息**
// 单独计算，不得换成其它序列化口径。
type DomainIDs struct {
	App      string
	Hardware string
	Ident    string
	OS       string
	Host     string
	Network  string
	Web      string
}

func isZero(v any) bool {
	return reflect.ValueOf(v).IsZero()
}

// upsertDomain 写入单个域行；id 为空或 payload 为零值时跳过（不建空行）。
func upsertDomain[T any](db *gorm.DB, id string, payload T) error {
	if id == "" || isZero(payload) {
		return nil
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
		return gorm.ErrInvalidData
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error
}

// Upsert 分表写入稳定域；跳过 HostLive / NetworkLive。返回主表 MD5。
// id 与 ids 均由调用方预计算（对 protobuf 稳定域 / 各域二进制做确定性 MD5，
// 与客户端算法一致），必填；id 为空视为非法入参。
func Upsert(db *gorm.DB, info *DeviceInfo, id string, ids DomainIDs) (string, error) {
	if info == nil || db == nil || id == "" {
		return "", gorm.ErrInvalidData
	}
	info.Normalize()

	for _, d := range []struct {
		id      string
		payload any
	}{
		{ids.App, info.App},
		{ids.Hardware, info.Hardware},
		{ids.Ident, info.ID},
		{ids.OS, info.OS},
		{ids.Host, info.Host},
		{ids.Network, info.Network},
		{ids.Web, info.Web},
	} {
		if err := upsertDomain(db, d.id, d.payload); err != nil {
			return "", err
		}
	}

	row := DeviceRow{
		Platform:   info.Platform,
		ClientKind: info.ClientKind,
		AppID:      ids.App,
		HardwareID: ids.Hardware,
		IdentID:    ids.Ident,
		OsID:       ids.OS,
		HostID:     ids.Host,
		NetworkID:  ids.Network,
		WebID:      ids.Web,
		Ext:        info.Ext,
		LastSeenAt: time.Now(),
	}
	row.ID = id
	// 同指纹重复上报只刷新最近活跃时间；域引用不变（id 已是稳定内容哈希）。
	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_seen_at"}),
	}).Create(&row).Error
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
