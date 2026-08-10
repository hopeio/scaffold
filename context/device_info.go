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

// 客户端形态（与 platform 正交：同为 android 可是 mobile App）。
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

// DeviceInfo 客户端环境信息：覆盖移动 / 桌面 / Web。
//
// 兼容旧版头解析字段（Device/OS/AppCode/AppVer/IP/Lng/Lat/Area/UserAgent）。
// 推荐客户端在 Device-Info 头直接传 JSON（与本结构体 json tag 对齐）；
// 亦支持旧 CSV：Device-Info=device,os ；App-Info=appCode,appVer。
type DeviceInfo struct {
	// —— 兼容旧版 / AccessDevice ——
	Device    string  `json:"device" gorm:"size:255"`
	OS        string  `json:"os" gorm:"size:255"`
	AppCode   string  `json:"appCode" gorm:"size:255"`
	AppVer    string  `json:"appVer" gorm:"size:255"`
	IP        net.IP  `json:"ip" gorm:"size:64"`
	Lng       float64 `json:"lng" gorm:"type:numeric(10,6)"`
	Lat       float64 `json:"lat" gorm:"type:numeric(10,6)"`
	Area      string  `json:"area" gorm:"size:255"`
	UserAgent string  `json:"userAgent" gorm:"size:512"`

	// —— 客户端形态 ——
	Platform   string `json:"platform" gorm:"size:32"`   // android|ios|macos|windows|linux|web
	ClientKind string `json:"clientKind" gorm:"size:32"` // mobile|desktop|web

	// —— 应用 ——
	AppName     string `json:"appName" gorm:"size:128"`
	AppBuild    string `json:"appBuild" gorm:"size:64"`
	PackageName string `json:"packageName" gorm:"size:255"`
	ServiceName string `json:"serviceName" gorm:"size:128"`

	// —— 设备硬件 ——
	Manufacturer string `json:"manufacturer" gorm:"size:128"`
	Brand        string `json:"brand" gorm:"size:128"`
	Model        string `json:"model" gorm:"size:128"`
	ModelName    string `json:"modelName" gorm:"size:255"`
	DeviceName   string `json:"deviceName" gorm:"size:255"`
	DeviceID     string `json:"deviceId" gorm:"size:255"` // IDFV / android id / machineId 等
	Product      string `json:"product" gorm:"size:128"`
	Board        string `json:"board" gorm:"size:128"`
	Hardware     string `json:"hardware" gorm:"size:128"`
	Fingerprint  string `json:"fingerprint" gorm:"size:512"`
	Bootloader   string `json:"bootloader" gorm:"size:128"`
	DisplayID    string `json:"displayId" gorm:"size:128"`
	IsPhysical   *bool  `json:"isPhysical,omitempty"`
	IsLowRam     *bool  `json:"isLowRam,omitempty"`

	// —— 操作系统 ——
	OSName          string `json:"osName" gorm:"size:64"`
	OSVersion       string `json:"osVersion" gorm:"size:64"`
	OSBuild         string `json:"osBuild" gorm:"size:128"`
	OSCodename      string `json:"osCodename" gorm:"size:64"`
	OSSDKInt        int    `json:"osSdkInt,omitempty"`
	OSSecurityPatch string `json:"osSecurityPatch" gorm:"size:32"`
	OSEdition       string `json:"osEdition" gorm:"size:128"`
	KernelVersion   string `json:"kernelVersion" gorm:"size:255"`
	Arch            string `json:"arch" gorm:"size:128"` // abi / machine / arch 列表

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

	// —— Web / 浏览器 ——
	Browser       string  `json:"browser" gorm:"size:64"`
	BrowserVer    string  `json:"browserVer" gorm:"size:64"`
	Engine        string  `json:"engine" gorm:"size:64"`
	EngineVer     string  `json:"engineVer" gorm:"size:64"`
	Vendor        string  `json:"vendor" gorm:"size:128"`
	Languages     string  `json:"languages" gorm:"size:255"` // 逗号分隔
	MaxTouchPts   int     `json:"maxTouchPoints,omitempty"`
	ScreenWidth   int     `json:"screenWidth,omitempty"`
	ScreenHeight  int     `json:"screenHeight,omitempty"`
	PixelRatio    float64 `json:"pixelRatio,omitempty"`
	ColorDepth    int     `json:"colorDepth,omitempty"`
	DeviceMemoryG float64 `json:"deviceMemoryGb,omitempty"`

	// —— 其它可扩展键（客户端自定义 / 未来字段）——
	Ext map[string]string `json:"ext,omitempty" gorm:"serializer:json"`
}

// Normalize 用扩展字段回填兼容字段，便于旧逻辑与 AccessDevice 映射。
func (d *DeviceInfo) Normalize() {
	if d == nil {
		return
	}
	if d.Device == "" {
		d.Device = firstNonEmpty(d.ModelName, d.Model, d.DeviceName, d.Brand, d.Product)
	}
	if d.OS == "" {
		d.OS = strings.TrimSpace(firstNonEmpty(d.OSName, d.Platform) + " " + d.OSVersion)
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
	if d.UserAgent == "" && d.Browser != "" {
		d.UserAgent = strings.TrimSpace(d.Browser + "/" + d.BrowserVer)
	}
}

func inferPlatform(d *DeviceInfo) string {
	s := strings.ToLower(strings.TrimSpace(d.OSName + " " + d.OS + " " + d.Platform))
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
	case d.Browser != "" || d.Engine != "":
		return PlatformWeb
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

// DeviceFromHeader extracts DeviceInfo from standard HTTP request headers.
// Device-Info 可为 JSON；否则按旧 CSV 解析，并合并 App-Info / Area / Location / UA / XFF。
func DeviceFromHeader(header http.Header) *DeviceInfo {
	rawDevice := header.Get(httpx.HeaderDeviceInfo)
	if info := tryParseDeviceJSON(rawDevice); info != nil {
		mergeAuxHeaders(info,
			header.Get(httpx.HeaderAppInfo),
			header.Get(httpx.HeaderArea),
			header.Get(httpx.HeaderLocation),
			header.Get(httpx.HeaderUserAgent),
			header.Get(httpx.HeaderXForwardedFor),
		)
		info.Normalize()
		return info
	}
	return Device(rawDevice, header.Get(httpx.HeaderAppInfo),
		header.Get(httpx.HeaderArea), header.Get(httpx.HeaderLocation),
		header.Get(httpx.HeaderUserAgent), header.Get(httpx.HeaderXForwardedFor))
}

func tryParseDeviceJSON(raw string) *DeviceInfo {
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

func mergeAuxHeaders(info *DeviceInfo, app, area, location, userAgent, ip string) {
	if info.AppCode == "" || info.AppVer == "" {
		code, ver := splitCSV2(app)
		if info.AppCode == "" {
			info.AppCode = code
		}
		if info.AppVer == "" {
			info.AppVer = ver
		}
	}
	if area != "" && info.Area == "" {
		info.Area, _ = url.QueryUnescape(area)
	}
	if location != "" && info.Lng == 0 && info.Lat == 0 {
		info.Lng, info.Lat = parseLatLng(location)
	}
	if userAgent != "" && info.UserAgent == "" {
		info.UserAgent = userAgent
	}
	if ip != "" && info.IP == nil {
		info.IP = net.ParseIP(firstForwardedIP(ip))
	}
}

// Device get device info
// device: device,os
// app: appCode,appVersion
// area: xxx
// location: lng,lat
func Device(device, app, area, location, userAgent, ip string) *DeviceInfo {
	info := new(DeviceInfo)
	unknow := true

	if device != "" {
		unknow = false
		info.Device, info.OS = splitCSV2(device)
	}
	if app != "" {
		unknow = false
		info.AppCode, info.AppVer = splitCSV2(app)
	}
	if area != "" {
		unknow = false
		info.Area, _ = url.QueryUnescape(area)
	}
	if location != "" {
		unknow = false
		info.Lng, info.Lat = parseLatLng(location)
	}
	if userAgent != "" {
		unknow = false
		info.UserAgent = userAgent
	}
	if ip != "" {
		unknow = false
		info.IP = net.ParseIP(firstForwardedIP(ip))
	}
	if unknow {
		return nil
	}
	info.Normalize()
	return info
}

func splitCSV2(s string) (a, b string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	parts := strings.SplitN(s, ",", 2)
	a = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		b = strings.TrimSpace(parts[1])
	}
	return a, b
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
