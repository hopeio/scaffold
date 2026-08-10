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

// DeviceInfo 客户端环境信息（按域拆分嵌套）。
// Device-Info 头传 JSON；服务端可补 network.ip / web.userAgent / network 地理头。
type DeviceInfo struct {
	Platform   string `json:"platform" gorm:"size:32"`   // android|ios|macos|windows|linux|web
	ClientKind string `json:"clientKind" gorm:"size:32"` // mobile|desktop|web

	App      DeviceAppInfo      `json:"app" gorm:"embedded;embeddedPrefix:app_"`
	Hardware DeviceHardwareInfo `json:"hardware" gorm:"embedded;embeddedPrefix:hw_"`
	IDs      DeviceIDInfo       `json:"ids" gorm:"embedded;embeddedPrefix:id_"`
	OS       DeviceOSInfo       `json:"os" gorm:"embedded;embeddedPrefix:os_"`
	Host     DeviceHostInfo     `json:"host" gorm:"embedded;embeddedPrefix:host_"`
	Network  DeviceNetworkInfo  `json:"network" gorm:"embedded;embeddedPrefix:net_"`
	Web      DeviceWebInfo      `json:"web" gorm:"embedded;embeddedPrefix:web_"`

	Ext map[string]string `json:"ext,omitempty" gorm:"serializer:json"`
}

// DeviceAppInfo 应用 / 包信息。
type DeviceAppInfo struct {
	Code        string `json:"code" gorm:"size:128"`
	Name        string `json:"name" gorm:"size:128"`
	Ver         string `json:"ver" gorm:"size:64"`
	Build       string `json:"build" gorm:"size:64"`
	PackageName string `json:"packageName" gorm:"size:255"` // Android applicationId
	BundleID    string `json:"bundleId" gorm:"size:255"`    // iOS / macOS
	Channel     string `json:"channel" gorm:"size:64"`
	ServiceName string `json:"serviceName" gorm:"size:128"`
}

// DeviceHardwareInfo 硬件画像。
// CPU/GPU 型号因平台而异：能采则填；iOS 通常只能靠机型反查，device_info_plus 本身不提供。
type DeviceHardwareInfo struct {
	Manufacturer string   `json:"manufacturer" gorm:"size:128"`
	Brand        string   `json:"brand" gorm:"size:128"`
	Model        string   `json:"model" gorm:"size:128"`
	ModelName    string   `json:"modelName" gorm:"size:255"`
	DeviceName   string   `json:"deviceName" gorm:"size:255"`
	Product      string   `json:"product" gorm:"size:128"`
	Board        string   `json:"board" gorm:"size:128"`
	Hardware     string   `json:"hardware" gorm:"size:128"` // Android Build.HARDWARE，常近似芯片平台
	Chipset      string   `json:"chipset" gorm:"size:128"`  // SoC，如 snapdragon 8 gen 3 / Apple M2
	Fingerprint  string   `json:"fingerprint" gorm:"size:512"`
	Bootloader   string   `json:"bootloader" gorm:"size:128"`
	DisplayID    string   `json:"displayId" gorm:"size:128"`
	SerialNo     string   `json:"serialNo" gorm:"size:128"`
	CPUVendor    string   `json:"cpuVendor" gorm:"size:64"`  // GenuineIntel / Apple / Qualcomm…
	CPUModel     string   `json:"cpuModel" gorm:"size:255"`  // 如 Apple M2 Pro / Intel Core i7-12700H
	CPUCores     int      `json:"cpuCores,omitempty"`        // 逻辑核；与 host.cpuCount 可同值
	GPUVendor    string   `json:"gpuVendor" gorm:"size:128"` // Apple / Qualcomm / NVIDIA…
	GPUModel     string   `json:"gpuModel" gorm:"size:255"`  // 如 Apple M2 / Adreno 740
	GPURenderer  string   `json:"gpuRenderer" gorm:"size:255"` // WebGL UNMASKED_RENDERER 等原始串
	IsPhysical   TriState `json:"isPhysical,omitempty" gorm:"type:smallint"`
	IsLowRam     TriState `json:"isLowRam,omitempty" gorm:"type:smallint"`
}

// DeviceIDInfo 设备 / 广告标识（能采到什么填什么）。
type DeviceIDInfo struct {
	DID          string   `json:"did" gorm:"size:128"`
	UUID         string   `json:"uuid" gorm:"size:128"`
	AndroidID    string   `json:"androidId" gorm:"size:128"`
	GAID         string   `json:"gaid" gorm:"size:128"`
	AAID         string   `json:"aaid" gorm:"size:128"`
	OAID         string   `json:"oaid" gorm:"size:128"`
	VAID         string   `json:"vaid" gorm:"size:128"`
	CAID         string   `json:"caid" gorm:"size:128"`
	IMEI         string   `json:"imei" gorm:"size:32"`
	IMEI2        string   `json:"imei2" gorm:"size:32"`
	MEID         string   `json:"meid" gorm:"size:32"`
	IDFA         string   `json:"idfa" gorm:"size:128"`
	IDFV         string   `json:"idfv" gorm:"size:128"`
	UDID         string   `json:"udid" gorm:"size:128"`
	OpenUDID     string   `json:"openUdid" gorm:"size:128"`
	GUID         string   `json:"guid" gorm:"size:128"`
	MAC          string   `json:"mac" gorm:"size:64"`
	WifiMAC      string   `json:"wifiMac" gorm:"size:64"`
	BluetoothMAC string   `json:"bluetoothMac" gorm:"size:64"`
	IDFATracking TriState `json:"idfaTracking,omitempty" gorm:"type:smallint"`
}

// DeviceOSInfo 操作系统。
type DeviceOSInfo struct {
	Name          string `json:"name" gorm:"size:64"`
	Version       string `json:"version" gorm:"size:64"`
	Build         string `json:"build" gorm:"size:128"`
	Codename      string `json:"codename" gorm:"size:64"`
	SDKInt        int    `json:"sdkInt,omitempty"`
	SecurityPatch string `json:"securityPatch" gorm:"size:32"`
	Edition       string `json:"edition" gorm:"size:128"`
	KernelVersion string `json:"kernelVersion" gorm:"size:255"`
	Arch          string `json:"arch" gorm:"size:128"`
}

// DeviceHostInfo 主机资源 / 语言区域快照。
type DeviceHostInfo struct {
	Locale       string `json:"locale" gorm:"size:64"`
	Language     string `json:"language" gorm:"size:32"`
	Languages    string `json:"languages" gorm:"size:255"`
	Timezone     string `json:"timezone" gorm:"size:64"`
	Hostname     string `json:"hostname" gorm:"size:255"`
	CPUCount     int    `json:"cpuCount,omitempty"`
	CPUFrequency int64  `json:"cpuFrequencyHz,omitempty"`
	RamMB        int64  `json:"ramMb,omitempty"`
	RamAvailMB   int64  `json:"ramAvailMb,omitempty"`
	DiskTotalB   int64  `json:"diskTotalBytes,omitempty"`
	DiskFreeB    int64  `json:"diskFreeBytes,omitempty"`
}

// DeviceNetworkInfo 网络 / 运营商 / 地理（含会话侧 IP）。
type DeviceNetworkInfo struct {
	IP          net.IP  `json:"ip" gorm:"size:64"`
	NetworkType string  `json:"networkType" gorm:"size:32"` // wifi|cellular|ethernet|vpn|unknown
	Carrier     string  `json:"carrier" gorm:"size:64"`
	ICCID       string  `json:"iccid" gorm:"size:32"`
	IMSI        string  `json:"imsi" gorm:"size:32"`
	Lng         float64 `json:"lng" gorm:"type:numeric(10,6)"`
	Lat         float64 `json:"lat" gorm:"type:numeric(10,6)"`
	Area        string  `json:"area" gorm:"size:255"`
}

// DeviceWebInfo 浏览器 / WebView。
type DeviceWebInfo struct {
	UserAgent     string   `json:"userAgent" gorm:"size:512"`
	Browser       string   `json:"browser" gorm:"size:64"`
	BrowserVer    string   `json:"browserVer" gorm:"size:64"`
	Engine        string   `json:"engine" gorm:"size:64"`
	EngineVer     string   `json:"engineVer" gorm:"size:64"`
	Vendor        string   `json:"vendor" gorm:"size:128"`
	MaxTouchPts   int      `json:"maxTouchPoints,omitempty"`
	ScreenWidth   int      `json:"screenWidth,omitempty"`
	ScreenHeight  int      `json:"screenHeight,omitempty"`
	PixelRatio    float64  `json:"pixelRatio,omitempty"`
	ColorDepth    int      `json:"colorDepth,omitempty"`
	DeviceMemoryG float64  `json:"deviceMemoryGb,omitempty"`
	CookieEnabled TriState `json:"cookieEnabled,omitempty" gorm:"type:smallint"`
	DoNotTrack    TriState `json:"doNotTrack,omitempty" gorm:"type:smallint"`
}

// DeviceInfoLite 业务表嵌入用的精简投影（行存）。
type DeviceInfoLite struct {
	Device    string  `json:"device" gorm:"size:255"`
	OS        string  `json:"os" gorm:"size:255"`
	AppCode   string  `json:"appCode" gorm:"size:255"`
	AppVer    string  `json:"appVer" gorm:"size:255"`
	IP        net.IP  `json:"ip" gorm:"size:64"`
	Lng       float64 `json:"lng" gorm:"type:numeric(10,6)"`
	Lat       float64 `json:"lat" gorm:"type:numeric(10,6)"`
	Area      string  `json:"area" gorm:"size:255"`
	UserAgent string  `json:"userAgent" gorm:"size:512"`
	DeviceNo  string  `json:"deviceNo" gorm:"size:128"`
}

// Lite 投影为精简行存结构。
func (d *DeviceInfo) Lite() DeviceInfoLite {
	if d == nil {
		return DeviceInfoLite{}
	}
	d.Normalize()
	return DeviceInfoLite{
		Device:    d.DisplayName(),
		OS:        d.OSDisplay(),
		AppCode:   d.App.Code,
		AppVer:    d.App.Ver,
		IP:        d.Network.IP,
		Lng:       d.Network.Lng,
		Lat:       d.Network.Lat,
		Area:      d.Network.Area,
		UserAgent: d.Web.UserAgent,
		DeviceNo:  d.PrimaryDeviceNo(),
	}
}

// DisplayName 人可读设备名。
func (d *DeviceInfo) DisplayName() string {
	if d == nil {
		return ""
	}
	h := d.Hardware
	return firstNonEmpty(h.ModelName, h.Model, h.DeviceName, h.Brand, h.Product)
}

// OSDisplay 人可读系统版本。
func (d *DeviceInfo) OSDisplay() string {
	if d == nil {
		return ""
	}
	return strings.TrimSpace(firstNonEmpty(d.OS.Name, d.Platform) + " " + d.OS.Version)
}

// PrimaryDeviceNo 优先业务 DID，再按常见标识回退。
func (d *DeviceInfo) PrimaryDeviceNo() string {
	if d == nil {
		return ""
	}
	id := d.IDs
	return firstNonEmpty(
		id.DID, id.IDFV, id.AndroidID, id.OAID, id.GAID, id.AAID,
		id.IDFA, id.GUID, id.UUID, id.UDID, id.OpenUDID, id.IMEI, id.MAC,
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
	if d.App.Ver == "" && d.App.Build != "" {
		d.App.Ver = d.App.Build
	}
	if d.App.PackageName == "" && d.App.BundleID != "" {
		d.App.PackageName = d.App.BundleID
	}
	if d.IDs.GAID == "" && d.IDs.AAID != "" {
		d.IDs.GAID = d.IDs.AAID
	}
	if d.IDs.AAID == "" && d.IDs.GAID != "" {
		d.IDs.AAID = d.IDs.GAID
	}
	if d.Web.UserAgent == "" && d.Web.Browser != "" {
		d.Web.UserAgent = strings.TrimSpace(d.Web.Browser + "/" + d.Web.BrowserVer)
	}
	if d.Host.Language == "" {
		if d.Host.Languages != "" {
			if i := strings.IndexByte(d.Host.Languages, ','); i >= 0 {
				d.Host.Language = strings.TrimSpace(d.Host.Languages[:i])
			} else {
				d.Host.Language = strings.TrimSpace(d.Host.Languages)
			}
		} else if d.Host.Locale != "" {
			d.Host.Language = strings.ReplaceAll(d.Host.Locale, "_", "-")
		}
	}
}

func inferPlatform(d *DeviceInfo) string {
	s := strings.ToLower(strings.TrimSpace(d.OS.Name + " " + d.Platform))
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
	case d.Web.Browser != "" || d.Web.Engine != "" || d.ClientKind == ClientKindWeb:
		return PlatformWeb
	case d.IDs.IDFV != "" || d.IDs.IDFA != "":
		return PlatformIOS
	case d.IDs.AndroidID != "" || d.IDs.OAID != "" || d.IDs.GAID != "" || d.IDs.IMEI != "":
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
	if area := header.Get(httpx.HeaderArea); area != "" && info.Network.Area == "" {
		info.Network.Area, _ = url.QueryUnescape(area)
	}
	if loc := header.Get(httpx.HeaderLocation); loc != "" && info.Network.Lng == 0 && info.Network.Lat == 0 {
		info.Network.Lng, info.Network.Lat = parseLatLng(loc)
	}
	if ua := header.Get(httpx.HeaderUserAgent); ua != "" && info.Web.UserAgent == "" {
		info.Web.UserAgent = ua
	}
	if xff := header.Get(httpx.HeaderXForwardedFor); xff != "" && info.Network.IP == nil {
		info.Network.IP = net.ParseIP(firstForwardedIP(xff))
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
		d.App.Code == "" && d.App.Ver == "" && d.Web.Browser == "" &&
		d.OS.Name == "" && d.OS.Version == "" && len(d.Ext) == 0 &&
		d.Web.UserAgent == "" && d.Network.Area == "" &&
		d.Network.Lng == 0 && d.Network.Lat == 0
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
