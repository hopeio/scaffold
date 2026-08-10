/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package context

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	httpx "github.com/hopeio/gox/net/http"
)

// 客户端形态（与 platform 正交）。
const (
	ClientKindMobile  = "mobile"
	ClientKindDesktop = "desktop"
	ClientKindWeb     = "web"
)

// 运行平台。
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformMacOS   = "macos"
	PlatformWindows = "windows"
	PlatformLinux   = "linux"
	PlatformWeb     = "web"
	PlatformUnknown = "unknown"
)

// TriState 三态开关：未采集 / 否 / 是（避免 bool 零值与「未填」混淆）。
type TriState int8

const (
	TriUnset TriState = 0
	TriFalse TriState = 1
	TriTrue  TriState = 2
)

func TriFromBool(v bool) TriState {
	if v {
		return TriTrue
	}
	return TriFalse
}

func (t TriState) IsTrue() bool  { return t == TriTrue }
func (t TriState) IsFalse() bool { return t == TriFalse }
func (t TriState) IsSet() bool   { return t == TriTrue || t == TriFalse }

// DeviceInfo 客户端环境信息：移动 / 桌面 / Web 全量字段。
// 客户端通过 Device-Info 头传 JSON（与 json tag 对齐）；服务端可补 IP / UA / 地理位置头。
type DeviceInfo struct {
	// —— 形态 ——
	Platform   string `json:"platform" gorm:"size:32"`   // android|ios|macos|windows|linux|web
	ClientKind string `json:"clientKind" gorm:"size:32"` // mobile|desktop|web

	// —— 应用 ——
	AppCode     string `json:"appCode" gorm:"size:128"`
	AppName     string `json:"appName" gorm:"size:128"`
	AppVer      string `json:"appVer" gorm:"size:64"`
	AppBuild    string `json:"appBuild" gorm:"size:64"`
	PackageName string `json:"packageName" gorm:"size:255"` // Android applicationId
	BundleID    string `json:"bundleId" gorm:"size:255"`    // iOS / macOS
	Channel     string `json:"channel" gorm:"size:64"`      // 分发渠道
	ServiceName string `json:"serviceName" gorm:"size:128"`

	// —— 硬件 ——
	Manufacturer string `json:"manufacturer" gorm:"size:128"`
	Brand        string `json:"brand" gorm:"size:128"`
	Model        string `json:"model" gorm:"size:128"`
	ModelName    string `json:"modelName" gorm:"size:255"`
	DeviceName   string `json:"deviceName" gorm:"size:255"`
	Product      string `json:"product" gorm:"size:128"`
	Board        string `json:"board" gorm:"size:128"`
	Hardware     string `json:"hardware" gorm:"size:128"`
	Fingerprint  string `json:"fingerprint" gorm:"size:512"`
	Bootloader   string `json:"bootloader" gorm:"size:128"`
	DisplayID    string `json:"displayId" gorm:"size:128"`
	SerialNo     string   `json:"serialNo" gorm:"size:128"`
	IsPhysical   TriState `json:"isPhysical,omitempty" gorm:"type:smallint"`
	IsLowRam     TriState `json:"isLowRam,omitempty" gorm:"type:smallint"`

	// —— 设备 / 广告标识（能采到什么填什么；权限受限时留空）——
	DID          string   `json:"did" gorm:"size:128"`          // 业务侧设备号
	UUID         string   `json:"uuid" gorm:"size:128"`         // 客户端自生成持久 UUID
	AndroidID    string   `json:"androidId" gorm:"size:128"`    // Settings.Secure.ANDROID_ID
	GAID         string   `json:"gaid" gorm:"size:128"`         // Google Advertising ID
	AAID         string   `json:"aaid" gorm:"size:128"`         // Android Advertising ID（常与 GAID 同；华为等别名）
	OAID         string   `json:"oaid" gorm:"size:128"`         // 中国移动智能终端联合会 MSA
	VAID         string   `json:"vaid" gorm:"size:128"`         // 开发者匿名设备标识
	CAID         string   `json:"caid" gorm:"size:128"`         // 中国广告协会互联网广告标识
	IMEI         string   `json:"imei" gorm:"size:32"`
	IMEI2        string   `json:"imei2" gorm:"size:32"` // 双卡第二卡
	MEID         string   `json:"meid" gorm:"size:32"`
	IDFA         string   `json:"idfa" gorm:"size:128"` // iOS Advertising Identifier
	IDFV         string   `json:"idfv" gorm:"size:128"` // identifierForVendor
	UDID         string   `json:"udid" gorm:"size:128"`
	OpenUDID     string   `json:"openUdid" gorm:"size:128"`
	GUID         string   `json:"guid" gorm:"size:128"` // Windows / 桌面机器 GUID
	MAC          string   `json:"mac" gorm:"size:64"`   // 主网卡 MAC
	WifiMAC      string   `json:"wifiMac" gorm:"size:64"`
	BluetoothMAC string   `json:"bluetoothMac" gorm:"size:64"`
	IDFATracking TriState `json:"idfaTracking,omitempty" gorm:"type:smallint"` // iOS ATT 是否授权

	// —— 操作系统 ——
	OSName          string `json:"osName" gorm:"size:64"`
	OSVersion       string `json:"osVersion" gorm:"size:64"`
	OSBuild         string `json:"osBuild" gorm:"size:128"`
	OSCodename      string `json:"osCodename" gorm:"size:64"`
	OSSDKInt        int    `json:"osSdkInt,omitempty"`
	OSSecurityPatch string `json:"osSecurityPatch" gorm:"size:32"`
	OSEdition       string `json:"osEdition" gorm:"size:128"`
	KernelVersion   string `json:"kernelVersion" gorm:"size:255"`
	Arch            string `json:"arch" gorm:"size:128"`

	// —— 主机资源快照 ——
	Locale       string `json:"locale" gorm:"size:64"`
	Timezone     string `json:"timezone" gorm:"size:64"`
	Hostname     string `json:"hostname" gorm:"size:255"`
	CPUCount     int    `json:"cpuCount,omitempty"`
	CPUFrequency int64  `json:"cpuFrequencyHz,omitempty"`
	RamMB        int64  `json:"ramMb,omitempty"`
	RamAvailMB   int64  `json:"ramAvailMb,omitempty"`
	DiskTotalB   int64  `json:"diskTotalBytes,omitempty"`
	DiskFreeB    int64  `json:"diskFreeBytes,omitempty"`

	// —— 网络 / 运营商 / 地理 ——
	IP          net.IP  `json:"ip" gorm:"size:64"`
	NetworkType string  `json:"networkType" gorm:"size:32"` // wifi|cellular|ethernet|vpn|unknown
	Carrier     string  `json:"carrier" gorm:"size:64"`
	ICCID       string  `json:"iccid" gorm:"size:32"`
	IMSI        string  `json:"imsi" gorm:"size:32"`
	Lng         float64 `json:"lng" gorm:"type:numeric(10,6)"`
	Lat         float64 `json:"lat" gorm:"type:numeric(10,6)"`
	Area        string  `json:"area" gorm:"size:255"`

	// —— Web / 浏览器 ——
	UserAgent     string  `json:"userAgent" gorm:"size:512"`
	Browser       string  `json:"browser" gorm:"size:64"`
	BrowserVer    string  `json:"browserVer" gorm:"size:64"`
	Engine        string  `json:"engine" gorm:"size:64"`
	EngineVer     string  `json:"engineVer" gorm:"size:64"`
	Vendor        string  `json:"vendor" gorm:"size:128"`
	Languages     string  `json:"languages" gorm:"size:255"`
	MaxTouchPts   int     `json:"maxTouchPoints,omitempty"`
	ScreenWidth   int     `json:"screenWidth,omitempty"`
	ScreenHeight  int     `json:"screenHeight,omitempty"`
	PixelRatio    float64 `json:"pixelRatio,omitempty"`
	ColorDepth    int     `json:"colorDepth,omitempty"`
	DeviceMemoryG float64  `json:"deviceMemoryGb,omitempty"`
	CookieEnabled TriState `json:"cookieEnabled,omitempty" gorm:"type:smallint"`
	DoNotTrack    TriState `json:"doNotTrack,omitempty" gorm:"type:smallint"`

	// —— 扩展 ——
	Ext map[string]string `json:"ext,omitempty" gorm:"serializer:json"`
}

// DisplayName 人可读设备名。
func (d *DeviceInfo) DisplayName() string {
	if d == nil {
		return ""
	}
	return firstNonEmpty(d.ModelName, d.Model, d.DeviceName, d.Brand, d.Product)
}

// OSDisplay 人可读系统版本。
func (d *DeviceInfo) OSDisplay() string {
	if d == nil {
		return ""
	}
	return strings.TrimSpace(firstNonEmpty(d.OSName, d.Platform) + " " + d.OSVersion)
}

// PrimaryDeviceNo 优先业务 DID，再按常见标识回退。
func (d *DeviceInfo) PrimaryDeviceNo() string {
	if d == nil {
		return ""
	}
	return firstNonEmpty(
		d.DID, d.IDFV, d.AndroidID, d.OAID, d.GAID, d.AAID,
		d.IDFA, d.GUID, d.UUID, d.UDID, d.OpenUDID, d.IMEI, d.MAC,
	)
}

// Normalize 补全 platform / clientKind 等可推导字段。
func (d *DeviceInfo) Normalize() {
	if d == nil {
		return
	}
	if d.Platform == "" {
		d.Platform = inferPlatform(d)
	}
	if d.ClientKind == "" {
		d.ClientKind = inferClientKind(d.Platform)
	}
	if d.AppVer == "" && d.AppBuild != "" {
		d.AppVer = d.AppBuild
	}
	if d.PackageName == "" && d.BundleID != "" {
		d.PackageName = d.BundleID
	}
	if d.GAID == "" && d.AAID != "" {
		d.GAID = d.AAID
	}
	if d.AAID == "" && d.GAID != "" {
		d.AAID = d.GAID
	}
	if d.UserAgent == "" && d.Browser != "" {
		d.UserAgent = strings.TrimSpace(d.Browser + "/" + d.BrowserVer)
	}
}

func inferPlatform(d *DeviceInfo) string {
	s := strings.ToLower(strings.TrimSpace(d.OSName + " " + d.Platform))
	switch {
	case strings.Contains(s, "android"):
		return PlatformAndroid
	case strings.Contains(s, "ios"), strings.Contains(s, "iphone"), strings.Contains(s, "ipad"):
		return PlatformIOS
	case strings.Contains(s, "macos"), strings.Contains(s, "darwin"), strings.Contains(s, "mac os"):
		return PlatformMacOS
	case strings.Contains(s, "windows"):
		return PlatformWindows
	case strings.Contains(s, "linux"):
		return PlatformLinux
	case d.Browser != "" || d.Engine != "" || d.ClientKind == ClientKindWeb:
		return PlatformWeb
	case d.IDFV != "" || d.IDFA != "":
		return PlatformIOS
	case d.AndroidID != "" || d.OAID != "" || d.GAID != "" || d.IMEI != "":
		return PlatformAndroid
	default:
		return PlatformUnknown
	}
}

func inferClientKind(platform string) string {
	switch platform {
	case PlatformAndroid, PlatformIOS:
		return ClientKindMobile
	case PlatformMacOS, PlatformWindows, PlatformLinux:
		return ClientKindDesktop
	case PlatformWeb:
		return ClientKindWeb
	default:
		return ""
	}
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

// DeviceFromHeader 从 Device-Info JSON 解析；并用 Area / Location / UA / XFF 补全。
func DeviceFromHeader(header http.Header) *DeviceInfo {
	info := parseDeviceJSON(header.Get(httpx.HeaderDeviceInfo))
	if info == nil {
		info = new(DeviceInfo)
	}
	if area := header.Get(httpx.HeaderArea); area != "" && info.Area == "" {
		info.Area, _ = url.QueryUnescape(area)
	}
	if loc := header.Get(httpx.HeaderLocation); loc != "" && info.Lng == 0 && info.Lat == 0 {
		info.Lng, info.Lat = parseLatLng(loc)
	}
	if ua := header.Get(httpx.HeaderUserAgent); ua != "" && info.UserAgent == "" {
		info.UserAgent = ua
	}
	if xff := header.Get(httpx.HeaderXForwardedFor); xff != "" && info.IP == nil {
		info.IP = net.ParseIP(firstForwardedIP(xff))
	}
	info.Normalize()
	if info.Empty() {
		return nil
	}
	return info
}

// Empty 是否几乎无有效客户端信息。
func (d *DeviceInfo) Empty() bool {
	if d == nil {
		return true
	}
	platEmpty := d.Platform == "" || d.Platform == PlatformUnknown
	return platEmpty &&
		d.DisplayName() == "" && d.PrimaryDeviceNo() == "" &&
		d.AppCode == "" && d.AppVer == "" && d.Browser == "" &&
		d.OSName == "" && d.OSVersion == "" && len(d.Ext) == 0 &&
		d.UserAgent == "" && d.Area == "" && d.Lng == 0 && d.Lat == 0
}

func parseDeviceJSON(raw string) *DeviceInfo {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	if unescaped, err := url.QueryUnescape(s); err == nil {
		s = strings.TrimSpace(unescaped)
	}
	if !strings.HasPrefix(s, "{") {
		return nil
	}
	info := new(DeviceInfo)
	if err := json.Unmarshal([]byte(s), info); err != nil {
		return nil
	}
	return info
}

func parseLatLng(location string) (lng, lat float64) {
	parts := strings.Split(location, ",")
	if len(parts) >= 1 {
		lng, _ = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	}
	if len(parts) >= 2 {
		lat, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	}
	return lng, lat
}

func firstForwardedIP(xff string) string {
	if i := strings.IndexByte(xff, ','); i >= 0 {
		return strings.TrimSpace(xff[:i])
	}
	return strings.TrimSpace(xff)
}
