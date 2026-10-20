/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License. See the LICENSE file in the project root for
 * more information.
 */

package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// A patch carries only what one upload actually reported. Two levels of absence
// matter and they mean different things:
//
//   - a nil domain pointer          -> that section was not reported at all;
//     nothing in it may be touched.
//   - a nil field inside a domain   -> that value could not be read this time;
//     it must not overwrite a value another client reported correctly.
//
// Within a domain, fields split into two overwrite classes. Getting this split
// wrong is what turned "user disabled ad tracking" into a permanently stale
// identifier, and what turned one failed read into a wiped device spec.

// IdentPatch: revocable identifiers. Present-but-empty is a real state (the user
// reset or denied the id), so it overwrites.
type IdentPatch struct {
	DID          *string
	UUID         *string
	AndroidID    *string
	GAID         *string
	AAID         *string
	OAID         *string
	VAID         *string
	CAID         *string
	IDFA         *string
	IDFV         *string
	OpenUDID     *string
	GUID         *string
	IDFATracking *TriState
}

// AppPatch feeds device_install. Versions describe an app on a device.
type AppPatch struct {
	Code, Name, Version, Build, PackageName, BundleID, Channel, ServiceName *string
}

// HardwarePatch / OSPatch / HostPatch / NetworkPatch / WebPatch are specs and
// descriptions: a zero here usually means "the read failed", so zero never
// overwrites a stored value.
type HardwarePatch struct {
	Manufacturer, Brand, Model, ModelName, DeviceName, Product, Board, Hardware, Chipset *string
	Fingerprint, Bootloader, DisplayID, SerialNo, CPUVendor, CPUModel                    *string
	GPUVendor, GPUModel, GPURenderer                                                     *string
	CPUCores                                                                             *int
	IsPhysical, IsLowRam                                                                 *TriState
}

type OSPatch struct {
	Name, Version, Build, Codename, SecurityPatch, Edition, KernelVersion, Arch, OSSkin, OSSkinVersion *string
	SDKInt                                                                                             *int
}

type HostPatch struct {
	Locale, Language, Languages, Timezone, Hostname *string
	CPUCount                                        *int
	CPUFrequencyHz, RamMb, DiskTotalBytes           *int64
}

type NetworkPatch struct{ Carrier, ICCID, IMSI *string }

type WebPatch struct {
	UserAgent, Browser, BrowserVersion, Engine, EngineVersion, Vendor *string
	MaxTouchPoints, ColorDepth                                        *int
	DeviceMemoryGb                                                    *float64
	CookieEnabled, DoNotTrack                                         *TriState
}

// Upload is one client report: only the domains it actually filled in.
type Upload struct {
	Platform    DeviceOS
	Channel     AppChannel
	FormFactor  FormFactor
	RuntimeHost RuntimeHost
	Ident       *IdentPatch
	App         *AppPatch
	Hardware    *HardwarePatch
	OS          *OSPatch
	Host        *HostPatch
	Network     *NetworkPatch
	Web         *WebPatch
	Ext         map[string]string
}

// put marks "reported, write it even if empty" — the revocable-identifier class.
func put[T any](m map[string]any, col string, v *T) {
	if v != nil {
		m[col] = *v
	}
}

// putNonZero marks "reported and usable" — the spec/description class. A zero
// here is treated as a failed read, which is the only reading consistent with
// what clients actually do when a platform has no such field.
func putNonZero[T comparable](m map[string]any, col string, v *T) {
	if v == nil {
		return
	}
	var zero T
	if *v == zero {
		return
	}
	m[col] = *v
}

func putTri(m map[string]any, col string, v *TriState) {
	// TriUnset is already the "unknown" encoding for this type, so an unset value
	// is not reported: writing it would erase a known true/false with "unknown".
	if v != nil && *v != TriUnset {
		m[col] = int8(*v)
	}
}

// identityOrder is sorted by how well the id survives normal events. Platform
// ids come first (they outlive app upgrades), resettable advertising ids next,
// the client-generated DID last: DID survives an upgrade but dies on
// reinstall/clear, so anchoring identity on it would mint a new device for every
// reinstall even when a stronger id was reported alongside it.
var identityOrder = []struct {
	pick func(*IdentPatch) *string
	kind KeyKind
}{
	{func(p *IdentPatch) *string { return p.IDFV }, KeyKindIDFV},
	{func(p *IdentPatch) *string { return p.AndroidID }, KeyKindAndroidID},
	{func(p *IdentPatch) *string { return p.OAID }, KeyKindOAID},
	{func(p *IdentPatch) *string { return p.GAID }, KeyKindGAID},
	{func(p *IdentPatch) *string { return p.AAID }, KeyKindAAID},
	{func(p *IdentPatch) *string { return p.VAID }, KeyKindVAID},
	{func(p *IdentPatch) *string { return p.CAID }, KeyKindCAID},
	{func(p *IdentPatch) *string { return p.OpenUDID }, KeyKindOpenUDID},
	{func(p *IdentPatch) *string { return p.GUID }, KeyKindGUID},
	{func(p *IdentPatch) *string { return p.UUID }, KeyKindUUID},
	{func(p *IdentPatch) *string { return p.DID }, KeyKindDID},
}

// CanonicalIdentity picks the device identity for an upload.
//
// It returns "" when nothing usable was reported. Callers must then refuse to
// write identity rather than fall back to a digest of the profile: a profile
// digest is exactly the old bug (switch language, get a new device), and an empty
// key would collide with every other unknown device on the unique index.
func CanonicalIdentity(p *IdentPatch) (string, KeyKind) {
	if p == nil {
		return "", KeyKindUnspecified
	}
	for _, e := range identityOrder {
		if v := e.pick(p); v != nil && *v != "" {
			return *v, e.kind
		}
	}
	return "", KeyKindUnspecified
}

// DeviceColumns builds the column map for the device row: identity, enums and
// every stable-domain column except the ones belonging to an app.
func (u *Upload) DeviceColumns() map[string]any {
	m := map[string]any{
		"platform":     int16(u.Platform),
		"client_kind":  int16(u.Channel),
		"form_factor":  int16(u.FormFactor),
		"runtime_host": int16(u.RuntimeHost),
	}
	if id := u.Ident; id != nil {
		// Revocable identifiers: present (including present-empty) overwrites.
		// They are not columns on device — they live in device_identifier — but
		// idfaTracking is a device-level state worth keeping.
		putTri(m, "idfa_tracking", id.IDFATracking)
	}
	if h := u.Hardware; h != nil {
		putNonZero(m, "manufacturer", h.Manufacturer)
		putNonZero(m, "brand", h.Brand)
		putNonZero(m, "model", h.Model)
		putNonZero(m, "model_name", h.ModelName)
		putNonZero(m, "device_name", h.DeviceName)
		putNonZero(m, "product", h.Product)
		putNonZero(m, "board", h.Board)
		putNonZero(m, "hardware", h.Hardware)
		putNonZero(m, "chipset", h.Chipset)
		putNonZero(m, "fingerprint", h.Fingerprint)
		putNonZero(m, "bootloader", h.Bootloader)
		putNonZero(m, "display_id", h.DisplayID)
		putNonZero(m, "serial_no", h.SerialNo)
		putNonZero(m, "cpu_vendor", h.CPUVendor)
		putNonZero(m, "cpu_model", h.CPUModel)
		putNonZero(m, "cpu_cores", h.CPUCores)
		putNonZero(m, "gpu_vendor", h.GPUVendor)
		putNonZero(m, "gpu_model", h.GPUModel)
		putNonZero(m, "gpu_renderer", h.GPURenderer)
		putTri(m, "is_physical", h.IsPhysical)
		putTri(m, "is_low_ram", h.IsLowRam)
	}
	if o := u.OS; o != nil {
		putNonZero(m, "os_name", o.Name)
		putNonZero(m, "os_version", o.Version)
		putNonZero(m, "os_build", o.Build)
		putNonZero(m, "os_codename", o.Codename)
		putNonZero(m, "os_sdk_int", o.SDKInt)
		putNonZero(m, "os_security_patch", o.SecurityPatch)
		putNonZero(m, "os_edition", o.Edition)
		putNonZero(m, "os_kernel_version", o.KernelVersion)
		putNonZero(m, "os_arch", o.Arch)
		putNonZero(m, "os_skin", o.OSSkin)
		putNonZero(m, "os_skin_version", o.OSSkinVersion)
	}
	if h := u.Host; h != nil {
		putNonZero(m, "locale", h.Locale)
		putNonZero(m, "language", h.Language)
		putNonZero(m, "languages", h.Languages)
		putNonZero(m, "timezone", h.Timezone)
		putNonZero(m, "hostname", h.Hostname)
		putNonZero(m, "cpu_count", h.CPUCount)
		putNonZero(m, "cpu_frequency", h.CPUFrequencyHz)
		putNonZero(m, "ram_mb", h.RamMb)
		putNonZero(m, "disk_total_byte", h.DiskTotalBytes)
	}
	if n := u.Network; n != nil {
		putNonZero(m, "carrier", n.Carrier)
		putNonZero(m, "iccid", n.ICCID)
		putNonZero(m, "imsi", n.IMSI)
	}
	if w := u.Web; w != nil {
		putNonZero(m, "user_agent", w.UserAgent)
		putNonZero(m, "browser", w.Browser)
		putNonZero(m, "browser_version", w.BrowserVersion)
		putNonZero(m, "engine", w.Engine)
		putNonZero(m, "engine_version", w.EngineVersion)
		putNonZero(m, "web_vendor", w.Vendor)
		// maxTouchPoints 0 is a real value (non-touch screen), so this one field is
		// "reported, write it" despite living in the spec class.
		put(m, "max_touch_pts", w.MaxTouchPoints)
		putNonZero(m, "color_depth", w.ColorDepth)
		putNonZero(m, "device_memory_g", w.DeviceMemoryGb)
		putTri(m, "cookie_enabled", w.CookieEnabled)
		putTri(m, "do_not_track", w.DoNotTrack)
	}
	if len(u.Ext) > 0 {
		m["ext"] = u.Ext
	}
	return m
}

// InstallColumns builds the device_install column map for one (device, app).
func (u *Upload) InstallColumns() (map[string]any, bool) {
	a := u.App
	if a == nil {
		return nil, false
	}
	m := map[string]any{"runtime_host": int16(u.RuntimeHost)}
	putNonZero(m, "app_code", a.Code)
	putNonZero(m, "app_name", a.Name)
	putNonZero(m, "app_version", a.Version)
	putNonZero(m, "app_build", a.Build)
	putNonZero(m, "package_name", a.PackageName)
	putNonZero(m, "bundle_id", a.BundleID)
	putNonZero(m, "app_channel", a.Channel)
	putNonZero(m, "service_name", a.ServiceName)
	return m, len(m) > 1
}

// DeviceIdentifierRow is one identifier seen on one device, kept over time so a
// reset or reinstalled id can be merged back instead of minting a new device.
type DeviceIdentifierRow struct {
	ID          uint64         `json:"id" gorm:"primaryKey"`
	DeviceID    uint64         `json:"deviceId" gorm:"index:idx_device_identifier_kind_value,unique"`
	Kind        KeyKind        `json:"kind" gorm:"column:kind;type:smallint;index:idx_device_identifier_kind_value,unique"`
	Value       string         `json:"value" gorm:"size:64;index:idx_device_identifier_kind_value,unique"`
	FirstSeenAt time.Time      `json:"firstSeenAt" gorm:"type:timestamptz(6)"`
	LastSeenAt  time.Time      `json:"lastSeenAt" gorm:"type:timestamptz(6);index"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"type:timestamptz(6);index"`
}

func (DeviceIdentifierRow) TableName() string { return "device_identifier" }

// DeviceChangeRow is the attribute audit trail: what a device's column used to
// be. Diff-only, not snapshots — a snapshot per report is how the content-
// addressed tables grew without bound.
//
// The comparison is string-normalised: this log is for humans, so a cosmetic
// formatting difference is harmless, while a wrong value would hide real changes.
type DeviceChangeRow struct {
	ID        uint64         `json:"id" gorm:"primaryKey"`
	DeviceID  uint64         `json:"deviceId" gorm:"index"`
	Scope     string         `json:"scope" gorm:"size:16;index"` // device | device_install
	FieldName string         `json:"field" gorm:"column:field;size:64"`
	OldValue  string         `json:"oldValue" gorm:"size:128;column:old_value"`
	NewValue  string         `json:"newValue" gorm:"size:128;column:new_value"`
	ChangedAt time.Time      `json:"changedAt" gorm:"type:timestamptz(6)"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"type:timestamptz(6);index"`
}

func (DeviceChangeRow) TableName() string { return "device_change" }

func normVal(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// ErrNoIdentity means the upload had no usable id at all. Callers must skip the v2
// write rather than invent a key: the previous behaviour (digest the whole profile)
// is what minted a new device every time the user switched language.
var ErrNoIdentity = errors.New("device: upload carries no usable identity")

type identPair struct {
	kind  KeyKind
	value string
}

// identPairs enumerates every identifier in the upload, including the one picked as
// canonical_key, so the whole set stays searchable for future merges.
func identPairs(p *IdentPatch) []identPair {
	if p == nil {
		return nil
	}
	out := make([]identPair, 0, len(identityOrder))
	for _, e := range identityOrder {
		if v := e.pick(p); v != nil && *v != "" {
			out = append(out, identPair{kind: e.kind, value: *v})
		}
	}
	return out
}

// extJSON serialises the extension map explicitly: the jsonb serializer lives on
// the struct field, and the patch path updates via a column map, so handing the raw
// map over would depend on the driver guessing an encoding.
func extJSON(m map[string]string) (string, bool) {
	if len(m) == 0 {
		return "", false
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// ApplyV2 writes one report into the v2 tables and returns the device id.
//
// Resolution order is deliberate: exact canonical_key hit, else merge via any
// identifier the device previously registered (a reset advertising id, a
// reinstalled DID on the same hardware), else a new device. Missing device rows are
// inserted whole; existing ones are patched column-by-column with only what this
// upload actually reported, and every genuinely changed column is logged.
func ApplyV2(db *gorm.DB, u *Upload, now time.Time) (uint64, error) {
	// Identity first, before anything else: an empty key on the unique index would
	// make every "identity unknown" upload collide into one shared device row, so
	// this must not depend on the rest of the arguments being well-formed.
	if u == nil {
		return 0, gorm.ErrInvalidData
	}
	key, kind := CanonicalIdentity(u.Ident)
	if key == "" {
		return 0, ErrNoIdentity
	}
	if db == nil {
		return 0, gorm.ErrInvalidData
	}
	pairs := identPairs(u.Ident)

	var dev DeviceRow
	if err := db.Where("canonical_key = ?", key).First(&dev).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
		dev = DeviceRow{}
	}
	if dev.ID == 0 {
		dev = mergeByIdentifier(db, pairs, dev.ID)
	}

	want := u.DeviceColumns()
	want["last_seen_at"] = now
	if s, ok := extJSON(u.Ext); ok {
		want["ext"] = s
	} else {
		delete(want, "ext")
	}

	if dev.ID == 0 {
		want["canonical_key"] = key
		want["key_kind"] = int8(kind)
		want["first_seen_at"] = now
		row := map[string]any{}
		for k, v := range want {
			row[k] = v
		}
		if err := db.Table("device").Create(row).Error; err != nil {
			return 0, err
		}
		if err := db.Where("canonical_key = ?", key).First(&dev).Error; err != nil {
			return 0, err
		}
	} else {
		old := map[string]any{}
		if err := db.Table("device").Where("id = ?", dev.ID).Take(&old).Error; err != nil {
			return 0, err
		}
		if ch := diffColumns(old, want); len(ch) > 0 {
			for i := range ch {
				ch[i].DeviceID = dev.ID
				ch[i].ChangedAt = now
				db.Create(&ch[i])
			}
		}
		if err := db.Model(&DeviceRow{}).Where("id = ?", dev.ID).Updates(want).Error; err != nil {
			return 0, err
		}
	}

	// Identifiers are recorded as observations, never moved: reassigning a value that
	// currently points at another device would silently merge two real machines, and
	// an over-merge cannot be undone later while a missed merge is just one more row
	// to look at.
	for _, p := range pairs {
		var ex DeviceIdentifierRow
		err := db.Where("kind = ? AND value = ?", p.kind, p.value).First(&ex).Error
		switch {
		case err == nil && ex.DeviceID == dev.ID:
			if e := db.Model(&DeviceIdentifierRow{}).Where("id = ?", ex.ID).
				Update("last_seen_at", now).Error; e != nil {
				return 0, e
			}
		case err == nil:
			// Owned by another device: leave it alone, do not merge on this basis.
		case errors.Is(err, gorm.ErrRecordNotFound):
			if e := db.Create(&DeviceIdentifierRow{
				DeviceID: dev.ID, Kind: p.kind, Value: p.value, FirstSeenAt: now, LastSeenAt: now,
			}).Error; e != nil {
				return 0, e
			}
		default:
			return 0, err
		}
	}

	// Without an app_code there is no (device, app) key to upsert on; the install
	// row is skipped rather than written under an empty code, which would make every
	// such upload collide into one shared row.
	if cols, ok := u.InstallColumns(); ok && cols["app_code"] != nil {
		code, _ := cols["app_code"].(string)
		var ex DeviceInstallRow
		q := db.Where("device_id = ? AND app_code = ?", dev.ID, code).First(&ex)
		cols["last_seen_at"] = now
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			cols["device_id"] = dev.ID
			cols["first_seen_at"] = now
			if err := db.Table("device_install").Create(cols).Error; err != nil {
				return 0, err
			}
		} else if q.Error != nil {
			return 0, q.Error
		} else if err := db.Model(&DeviceInstallRow{}).Where("id = ?", ex.ID).Updates(cols).Error; err != nil {
			return 0, err
		}
	}
	return dev.ID, nil
}

// mergeByIdentifier resolves a device that already knows one of these ids. When
// several match, the most recently seen wins: that is the machine the user is most
// likely holding now.
func mergeByIdentifier(db *gorm.DB, pairs []identPair, _ uint64) DeviceRow {
	if len(pairs) == 0 {
		return DeviceRow{}
	}
	var ids []uint64
	for _, p := range pairs {
		var row DeviceIdentifierRow
		if err := db.Where("kind = ? AND value = ?", p.kind, p.value).First(&row).Error; err == nil && row.DeviceID != 0 {
			ids = append(ids, row.DeviceID)
		}
	}
	if len(ids) == 0 {
		return DeviceRow{}
	}
	var dev DeviceRow
	if err := db.Where("id IN ?", ids).Order("last_seen_at desc").First(&dev).Error; err != nil {
		return DeviceRow{}
	}
	return dev
}

// BindOwner records that userID used this device now. Ownership is append-only:
// the previous live binding is closed rather than rewritten, so a second-hand phone
// keeps both owners' history. Re-binding the same user is a no-op on the row and
// only refreshes nothing (last_seen_at already moved).
func BindOwner(db *gorm.DB, deviceID, userID uint64, source string, now time.Time) error {
	if db == nil || deviceID == 0 || userID == 0 {
		return nil
	}
	var live DeviceOwnerRow
	err := db.Where("device_id = ? AND user_id = ? AND unbound_at IS NULL", deviceID, userID).First(&live).Error
	if err == nil {
		return nil // already bound to this user; history stays as it is
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := db.Model(&DeviceOwnerRow{}).
		Where("device_id = ? AND unbound_at IS NULL", deviceID).
		Update("unbound_at", now).Error; err != nil {
		return err
	}
	return db.Create(&DeviceOwnerRow{DeviceID: deviceID, UserID: userID, BoundAt: now, Source: source}).Error
}

// diffColumns compares a desired column map against the old row as loaded from
// the database, returning only genuinely changed fields.
func diffColumns(old map[string]any, want map[string]any) []DeviceChangeRow {
	var out []DeviceChangeRow
	for col, nv := range want {
		ov := normVal(old[col])
		nnv := normVal(nv)
		if ov == nnv {
			continue
		}
		out = append(out, DeviceChangeRow{Scope: "device", FieldName: col, OldValue: ov, NewValue: nnv})
	}
	return out
}
