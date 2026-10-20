/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License. See the LICENSE file in the project root for
 * more information.
 */

package device

import (
	"time"

	"gorm.io/gorm"
)

// Storage: one row per physical device, one row per (device, app), attributes as
// plain columns, extension key-values as jsonb. There is no older layout beside it:
// the content-addressed domain tables and the profile-digest device key they hung on
// are gone, because digesting the profile made a language switch or an app upgrade
// look like a new device.
//
// Identity vs profile: identity is canonical_key below, derived server-side from the
// strongest device id present in the report, falling back to the client-generated DID.
// It is a heuristic, not a credential — headers are client-supplied and can be
// forged, so quota and "my devices" counts must not be treated as security
// boundaries (the product deliberately does not issue a server-signed device
// token). The volatile profile columns never feed the identity.

// KeyKind says what canonical_key actually is, because "an id string" without a
// kind cannot be compared, migrated or trusted at the right strength.
type KeyKind int8

const (
	KeyKindUnspecified KeyKind = 0
	KeyKindIDFV        KeyKind = 1
	KeyKindAndroidID   KeyKind = 2
	KeyKindOAID        KeyKind = 3
	KeyKindGAID        KeyKind = 4
	KeyKindAAID        KeyKind = 5
	KeyKindVAID        KeyKind = 6
	KeyKindCAID        KeyKind = 7
	KeyKindOpenUDID    KeyKind = 8
	KeyKindGUID        KeyKind = 9
	KeyKindUUID        KeyKind = 10
	KeyKindDID         KeyKind = 11 // client-generated, survives upgrade, lost on reinstall/clear
)

// DeviceRow is one physical device. A model/OS/app change updates this row in
// place; it never creates a second row (that was the core defect of the md5
// digest key). Two owners over time are two rows in DeviceOwnerRow, not two
// devices.
type DeviceRow struct {
	ID uint64 `json:"id" gorm:"primaryKey"`

	// Identity. uniqueIndex: exactly one row per device.
	CanonicalKey string  `json:"canonicalKey" gorm:"uniqueIndex;size:64"`
	KeyKind      KeyKind `json:"keyKind" gorm:"type:smallint"`

	Platform    DeviceOS    `json:"platform" gorm:"type:smallint"`
	Channel     AppChannel  `json:"channel" gorm:"column:client_kind;type:smallint"`
	FormFactor  FormFactor  `json:"formFactor" gorm:"column:form_factor;type:smallint"`
	RuntimeHost RuntimeHost `json:"runtimeHost" gorm:"column:runtime_host;type:smallint"`

	// hardware
	Manufacturer string   `json:"manufacturer" gorm:"size:64;index"`
	Brand        string   `json:"brand" gorm:"size:64"`
	Model        string   `json:"model" gorm:"size:64;index"`
	ModelName    string   `json:"modelName" gorm:"size:128"`
	DeviceName   string   `json:"deviceName" gorm:"size:128"`
	Product      string   `json:"product" gorm:"size:64"`
	Board        string   `json:"board" gorm:"size:64"`
	Hardware     string   `json:"hardware" gorm:"size:64"`
	Chipset      string   `json:"chipset" gorm:"size:64"`
	Fingerprint  string   `json:"fingerprint" gorm:"size:128"` // build fingerprint: expected to change on every OTA
	Bootloader   string   `json:"bootloader" gorm:"size:64"`
	DisplayID    string   `json:"displayId" gorm:"size:64"`
	SerialNo     string   `json:"serialNo" gorm:"size:64"`
	CPUVendor    string   `json:"cpuVendor" gorm:"size:64"`
	CPUModel     string   `json:"cpuModel" gorm:"size:128"`
	CPUCores     int      `json:"cpuCores"`
	GPUVendor    string   `json:"gpuVendor" gorm:"size:64"`
	GPUModel     string   `json:"gpuModel" gorm:"size:128"`
	GPURenderer  string   `json:"gpuRenderer" gorm:"size:128"`
	IsPhysical   TriState `json:"isPhysical" gorm:"type:smallint"`
	IsLowRam     TriState `json:"isLowRam" gorm:"type:smallint"`

	// os
	OSName          string `json:"osName" gorm:"size:32;column:os_name;index"`
	OSVersion       string `json:"osVersion" gorm:"size:32;column:os_version"`
	OSBuild         string `json:"osBuild" gorm:"size:64;column:os_build"`
	OSCodename      string `json:"osCodename" gorm:"size:32;column:os_codename"`
	OSSDKInt        int    `json:"sdkInt" gorm:"column:os_sdk_int"`
	OSSecurityPatch string `json:"securityPatch" gorm:"size:32;column:os_security_patch"`
	OSEdition       string `json:"edition" gorm:"size:64;column:os_edition"`
	OSKernelVersion string `json:"kernelVersion" gorm:"size:64;column:os_kernel_version"`
	OSArch          string `json:"arch" gorm:"size:16;column:os_arch"`
	OSSkin          string `json:"osSkin" gorm:"size:32;column:os_skin;index"`
	OSSkinVersion   string `json:"osSkinVersion" gorm:"size:32;column:os_skin_version"`

	// host
	Locale        string `json:"locale" gorm:"size:32"`
	Language      string `json:"language" gorm:"size:16;index"`
	Languages     string `json:"languages" gorm:"size:128"`
	Timezone      string `json:"timezone" gorm:"size:64"`
	Hostname      string `json:"hostname" gorm:"size:128"`
	CPUCount      int    `json:"cpuCount"`
	CPUFrequency  int64  `json:"cpuFrequencyHz"`
	RamMB         int64  `json:"ramMb"`
	DiskTotalByte int64  `json:"diskTotalBytes"`

	// network
	Carrier string `json:"carrier" gorm:"size:64"`
	ICCID   string `json:"iccid" gorm:"size:32"`
	IMSI    string `json:"imsi" gorm:"size:32"`

	// web
	UserAgent     string   `json:"userAgent" gorm:"size:255"`
	Browser       string   `json:"browser" gorm:"size:32"`
	BrowserVer    string   `json:"browserVersion" gorm:"size:32;column:browser_version"`
	Engine        string   `json:"engine" gorm:"size:32"`
	EngineVer     string   `json:"engineVersion" gorm:"size:32;column:engine_version"`
	WebVendor     string   `json:"webVendor" gorm:"size:64;column:web_vendor"`
	MaxTouchPts   int      `json:"maxTouchPoints"`
	ColorDepth    int      `json:"colorDepth"`
	DeviceMemoryG float64  `json:"deviceMemoryGb"`
	CookieEnabled TriState `json:"cookieEnabled" gorm:"type:smallint"`
	DoNotTrack    TriState `json:"doNotTrack" gorm:"type:smallint"`

	// display metrics: they change with window size, zoom and external monitors, so
	// they are patched in place and must never feed identity (the old layout kept them
	// in a separate table for the same reason).
	ScreenWidth  int     `json:"screenWidth"`
	ScreenHeight int     `json:"screenHeight"`
	PixelRatio   float64 `json:"pixelRatio"`

	// Extension key-values only: anything with a known shape is a column above.
	Ext map[string]string `json:"ext,omitempty" gorm:"serializer:json"`

	FirstSeenAt time.Time      `json:"firstSeenAt" gorm:"type:timestamptz(6)"`
	LastSeenAt  time.Time      `json:"lastSeenAt" gorm:"type:timestamptz(6);index"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"type:timestamptz(6);index"`
}

func (DeviceRow) TableName() string { return "device" }

// DeviceInstallRow is one (device, app) pair: versions belong to an app on a
// device, not to the device. Collapsing them into DeviceRow made a mini-program
// report overwrite the App's version and vice versa.
type DeviceInstallRow struct {
	ID          uint64         `json:"id" gorm:"primaryKey"`
	DeviceID    uint64         `json:"deviceId" gorm:"index"`
	AppCode     string         `json:"appCode" gorm:"size:32"`
	AppName     string         `json:"appName" gorm:"size:64;column:app_name"`
	Version     string         `json:"version" gorm:"size:32;column:app_version"`
	Build       string         `json:"build" gorm:"size:32;column:app_build"`
	PackageName string         `json:"packageName" gorm:"size:128"`
	BundleID    string         `json:"bundleId" gorm:"size:128"`
	Channel     string         `json:"channel" gorm:"size:32;column:app_channel"`
	ServiceName string         `json:"serviceName" gorm:"size:64"`
	RuntimeHost RuntimeHost    `json:"runtimeHost" gorm:"column:runtime_host;type:smallint"`
	FirstSeenAt time.Time      `json:"firstSeenAt" gorm:"type:timestamptz(6)"`
	LastSeenAt  time.Time      `json:"lastSeenAt" gorm:"type:timestamptz(6)"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"type:timestamptz(6);index"`
}

func (DeviceInstallRow) TableName() string { return "device_install" }

// DeviceOwnerRow records ownership over time. It is never updated to a new owner
// and never deleted: a second-hand phone must not erase the previous owner's
// history, and quota counts only live bindings (unbound_at IS NULL and seen
// within the active window).
type DeviceOwnerRow struct {
	ID        uint64         `json:"id" gorm:"primaryKey"`
	DeviceID  uint64         `json:"deviceId" gorm:"index"`
	UserID    uint64         `json:"userId" gorm:"index"`
	BoundAt   time.Time      `json:"boundAt" gorm:"type:timestamptz(6)"`
	UnboundAt *time.Time     `json:"unboundAt,omitempty" gorm:"type:timestamptz(6)"`
	Source    string         `json:"source" gorm:"size:32"` // login | manual | migrate
	DeletedAt gorm.DeletedAt `json:"-" gorm:"type:timestamptz(6);index"`
}

func (DeviceOwnerRow) TableName() string { return "device_owner" }

// Indexes backs the tables above, run by the caller after migrating (GORM tags
// cannot express a partial unique, and that is exactly what "one live binding per
// device per user" needs). Kept next to the rows so the DDL cannot drift from the
// struct it indexes.
var Indexes = []string{
	// One current-state row per (device, app): it is patched in place instead of
	// growing a new row per version change.
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_device_install ON "device_install" (device_id, app_code) WHERE deleted_at IS NULL`,
	// One live binding per (device, user); closed history rows stay unconstrained.
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_device_owner_live ON "device_owner" (device_id, user_id) WHERE unbound_at IS NULL AND deleted_at IS NULL`,
	// Merge lookup: find which device already claims a given (kind, value) id.
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_device_identifier ON "device_identifier" (kind, value) WHERE deleted_at IS NULL`,
}
