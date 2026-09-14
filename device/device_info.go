/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package device

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	httpx "github.com/hopeio/gox/net/http"
)

// DeviceOS 操作系统（与 AppChannel / FormFactor 正交；不含 web）。数值对齐 common.DeviceOS。
// 仅表示底层 OS/runtime，不隐含任何形态：Android 既可是手机也可是平板/TV/车载/嵌入式。
type DeviceOS int8

const (
	DeviceOSUnspecified DeviceOS = 0
	DeviceOSAndroid     DeviceOS = 1
	DeviceOSIos         DeviceOS = 2
	DeviceOSMacos       DeviceOS = 3
	DeviceOSWindows     DeviceOS = 4
	DeviceOSLinux       DeviceOS = 5
	DeviceOSUnknown     DeviceOS = 6
	DeviceOSHarmonyOS   DeviceOS = 7 // 鸿蒙
	DeviceOSRTOS        DeviceOS = 8 // 嵌入式 RTOS（底座，非形态）
)

func (p DeviceOS) String() string {
	switch p {
	case DeviceOSAndroid:
		return "android"
	case DeviceOSIos:
		return "ios"
	case DeviceOSMacos:
		return "macos"
	case DeviceOSWindows:
		return "windows"
	case DeviceOSLinux:
		return "linux"
	case DeviceOSUnknown:
		return "unknown"
	case DeviceOSHarmonyOS:
		return "harmonyos"
	case DeviceOSRTOS:
		return "rtos"
	default:
		return ""
	}
}

// ParseDeviceOS 解析 Platform-Info / JSON 线格式（短名或数字）。数值对齐 common.DeviceOS。
func ParseDeviceOS(s string) DeviceOS {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "0", "unspecified":
		return DeviceOSUnspecified
	case "1", "android":
		return DeviceOSAndroid
	case "2", "ios":
		return DeviceOSIos
	case "3", "macos":
		return DeviceOSMacos
	case "4", "windows":
		return DeviceOSWindows
	case "5", "linux":
		return DeviceOSLinux
	case "6", "unknown":
		return DeviceOSUnknown
	case "7", "harmonyos", "hongmeng":
		return DeviceOSHarmonyOS
	case "8", "rtos":
		return DeviceOSRTOS
	default:
		if n, err := strconv.Atoi(s); err == nil {
			return DeviceOS(n)
		}
		return DeviceOSUnspecified
	}
}

func (p DeviceOS) Value() (driver.Value, error) { return int64(p), nil }

func (p *DeviceOS) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*p = DeviceOS(v)
	case int32:
		*p = DeviceOS(v)
	case int:
		*p = DeviceOS(v)
	case []byte:
		*p = ParseDeviceOS(string(v))
	case string:
		*p = ParseDeviceOS(v)
	case nil:
		*p = DeviceOSUnspecified
	default:
		return fmt.Errorf("device: cannot scan %T into DeviceOS", src)
	}
	return nil
}

func (p DeviceOS) MarshalText() ([]byte, error) { return []byte(p.String()), nil }

func (p *DeviceOS) UnmarshalText(b []byte) error {
	*p = ParseDeviceOS(string(b))
	return nil
}

func (p DeviceOS) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(p))
}

func (p *DeviceOS) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*p = ParseDeviceOS(s)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*p = DeviceOS(n)
	return nil
}

// AppChannel 交付渠道（与 DeviceOS / FormFactor 正交；非 AppConfig 产品通道）。
// 原生 App / 小程序 / 浏览器 Web / 内嵌 WebView 互不重叠，与 OS 任意组合。
type AppChannel int8

const (
	AppChannelUnspecified AppChannel = 0
	AppChannelNative      AppChannel = 1 // 原生 App
	AppChannelMiniProgram AppChannel = 2 // 微信/支付宝/抖音等小程序
	AppChannelWeb         AppChannel = 3 // 浏览器 Web
	AppChannelWebview     AppChannel = 4 // 内嵌 WebView（宿主见 RuntimeHost）
)

func (k AppChannel) String() string {
	switch k {
	case AppChannelNative:
		return "native"
	case AppChannelMiniProgram:
		return "miniprogram"
	case AppChannelWeb:
		return "web"
	case AppChannelWebview:
		return "webview"
	default:
		return ""
	}
}

// ParseAppChannel 解析 Platform-Info / JSON 线格式（短名或数字）。
// 兼容旧客户端上送的 form-factor 词（mobile/tablet/desktop）→ native，降低切换期丢数据。
func ParseAppChannel(s string) AppChannel {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "0", "unspecified":
		return AppChannelUnspecified
	case "1", "native", "mobile", "tablet", "desktop":
		return AppChannelNative
	case "2", "miniprogram", "mini-program", "miniapp":
		return AppChannelMiniProgram
	case "3", "web":
		return AppChannelWeb
	case "4", "webview":
		return AppChannelWebview
	default:
		if n, err := strconv.Atoi(s); err == nil {
			return AppChannel(n)
		}
		return AppChannelUnspecified
	}
}

func (k AppChannel) Value() (driver.Value, error) { return int64(k), nil }

func (k *AppChannel) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*k = AppChannel(v)
	case int32:
		*k = AppChannel(v)
	case int:
		*k = AppChannel(v)
	case []byte:
		*k = ParseAppChannel(string(v))
	case string:
		*k = ParseAppChannel(v)
	case nil:
		*k = AppChannelUnspecified
	default:
		return fmt.Errorf("device: cannot scan %T into AppChannel", src)
	}
	return nil
}

func (k AppChannel) MarshalText() ([]byte, error) { return []byte(k.String()), nil }

func (k *AppChannel) UnmarshalText(b []byte) error {
	*k = ParseAppChannel(string(b))
	return nil
}

func (k AppChannel) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(k))
}

func (k *AppChannel) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*k = ParseAppChannel(s)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*k = AppChannel(n)
	return nil
}

// FormFactor 设备形态（与 DeviceOS / AppChannel 正交）。
// 不推断、由客户端显式上报：Android/iOS 不一定是手机，Linux 不一定是桌面。
type FormFactor int8

const (
	FormFactorUnspecified FormFactor = 0
	FormFactorPhone       FormFactor = 1
	FormFactorTablet      FormFactor = 2
	FormFactorDesktop     FormFactor = 3
	FormFactorLaptop      FormFactor = 4
	FormFactorTV          FormFactor = 5
	FormFactorWearable    FormFactor = 6
	FormFactorEmbedded    FormFactor = 7 // 工控屏/机顶盒/POS 等嵌入式设备
	FormFactorCar         FormFactor = 8 // 车机
	FormFactorKiosk       FormFactor = 9 // 自助终端
	FormFactorUnknown     FormFactor = 10
)

func (f FormFactor) String() string {
	switch f {
	case FormFactorPhone:
		return "phone"
	case FormFactorTablet:
		return "tablet"
	case FormFactorDesktop:
		return "desktop"
	case FormFactorLaptop:
		return "laptop"
	case FormFactorTV:
		return "tv"
	case FormFactorWearable:
		return "wearable"
	case FormFactorEmbedded:
		return "embedded"
	case FormFactorCar:
		return "car"
	case FormFactorKiosk:
		return "kiosk"
	case FormFactorUnknown:
		return "unknown"
	default:
		return ""
	}
}

// ParseFormFactor 解析 JSON 线格式（短名或数字）。数值对齐 common.FormFactor。
func ParseFormFactor(s string) FormFactor {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "0", "unspecified":
		return FormFactorUnspecified
	case "1", "phone":
		return FormFactorPhone
	case "2", "tablet":
		return FormFactorTablet
	case "3", "desktop":
		return FormFactorDesktop
	case "4", "laptop":
		return FormFactorLaptop
	case "5", "tv":
		return FormFactorTV
	case "6", "wearable":
		return FormFactorWearable
	case "7", "embedded":
		return FormFactorEmbedded
	case "8", "car":
		return FormFactorCar
	case "9", "kiosk":
		return FormFactorKiosk
	case "10", "unknown":
		return FormFactorUnknown
	default:
		if n, err := strconv.Atoi(s); err == nil {
			return FormFactor(n)
		}
		return FormFactorUnspecified
	}
}

func (f FormFactor) Value() (driver.Value, error) { return int64(f), nil }

func (f *FormFactor) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*f = FormFactor(v)
	case int32:
		*f = FormFactor(v)
	case int:
		*f = FormFactor(v)
	case []byte:
		*f = ParseFormFactor(string(v))
	case string:
		*f = ParseFormFactor(v)
	case nil:
		*f = FormFactorUnspecified
	default:
		return fmt.Errorf("device: cannot scan %T into FormFactor", src)
	}
	return nil
}

func (f FormFactor) MarshalText() ([]byte, error) { return []byte(f.String()), nil }

func (f *FormFactor) UnmarshalText(b []byte) error {
	*f = ParseFormFactor(string(b))
	return nil
}

func (f FormFactor) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(f))
}

func (f *FormFactor) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = ParseFormFactor(s)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = FormFactor(n)
	return nil
}

// RuntimeHost 小程序 / WebView 宿主（仅 AppChannel=MiniProgram/Webview 有意义）。
// 区分各平台小程序（微信/支付宝/抖音/百度/QQ/快手/京东/飞书/小红书/鸿蒙）与独立 App 内嵌 WebView。
// 宿主枚举数值对齐 proto common.RuntimeHost；uniapp 平台串（mp-weixin 等）可直接解析。
type RuntimeHost int8

const (
	RuntimeHostUnspecified RuntimeHost = 0
	RuntimeHostWeChat      RuntimeHost = 1  // 微信小程序（mp-weixin）
	RuntimeHostAlipay      RuntimeHost = 2  // 支付宝小程序（mp-alipay）
	RuntimeHostDouyin      RuntimeHost = 3  // 抖音/字节小程序（mp-toutiao）
	RuntimeHostBaidu       RuntimeHost = 4  // 百度小程序（mp-baidu）
	RuntimeHostApp         RuntimeHost = 5  // 独立 App 内嵌的 WebView
	RuntimeHostQQ          RuntimeHost = 6  // QQ 小程序（mp-qq）
	RuntimeHostKuaishou    RuntimeHost = 7  // 快手小程序（mp-kuaishou）
	RuntimeHostJD          RuntimeHost = 8  // 京东小程序（mp-jd）
	RuntimeHostLark        RuntimeHost = 9  // 飞书小程序（mp-lark）
	RuntimeHostXHS         RuntimeHost = 10 // 小红书小程序（mp-xhs）
	RuntimeHostHarmony     RuntimeHost = 11 // 鸿蒙小程序（mp-harmony）
	RuntimeHostQuickApp    RuntimeHost = 12 // 快应用（quickapp-webview）
)

func (h RuntimeHost) String() string {
	switch h {
	case RuntimeHostWeChat:
		return "wechat"
	case RuntimeHostAlipay:
		return "alipay"
	case RuntimeHostDouyin:
		return "douyin"
	case RuntimeHostBaidu:
		return "baidu"
	case RuntimeHostApp:
		return "app"
	case RuntimeHostQQ:
		return "qq"
	case RuntimeHostKuaishou:
		return "kuaishou"
	case RuntimeHostJD:
		return "jd"
	case RuntimeHostLark:
		return "lark"
	case RuntimeHostXHS:
		return "xhs"
	case RuntimeHostHarmony:
		return "harmony"
	case RuntimeHostQuickApp:
		return "quickapp"
	default:
		return ""
	}
}

// ParseRuntimeHost 解析 JSON 线格式（短名或数字）。数值对齐 common.RuntimeHost。
func ParseRuntimeHost(s string) RuntimeHost {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "0", "unspecified":
		return RuntimeHostUnspecified
	case "1", "wechat", "wx", "mp-weixin":
		return RuntimeHostWeChat
	case "2", "alipay", "mp-alipay":
		return RuntimeHostAlipay
	case "3", "douyin", "toutiao", "mp-toutiao":
		return RuntimeHostDouyin
	case "4", "baidu", "mp-baidu":
		return RuntimeHostBaidu
	case "5", "app", "standalone", "standaloneapp":
		return RuntimeHostApp
	case "6", "qq", "mp-qq":
		return RuntimeHostQQ
	case "7", "kuaishou", "mp-kuaishou":
		return RuntimeHostKuaishou
	case "8", "jd", "mp-jd":
		return RuntimeHostJD
	case "9", "lark", "feishu", "mp-lark":
		return RuntimeHostLark
	case "10", "xhs", "xiaohongshu", "mp-xhs":
		return RuntimeHostXHS
	case "11", "harmony", "mp-harmony":
		return RuntimeHostHarmony
	case "12", "quickapp", "quickapp-webview":
		return RuntimeHostQuickApp
	default:
		if n, err := strconv.Atoi(s); err == nil {
			return RuntimeHost(n)
		}
		return RuntimeHostUnspecified
	}
}

func (h RuntimeHost) Value() (driver.Value, error) { return int64(h), nil }

func (h *RuntimeHost) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*h = RuntimeHost(v)
	case int32:
		*h = RuntimeHost(v)
	case int:
		*h = RuntimeHost(v)
	case []byte:
		*h = ParseRuntimeHost(string(v))
	case string:
		*h = ParseRuntimeHost(v)
	case nil:
		*h = RuntimeHostUnspecified
	default:
		return fmt.Errorf("device: cannot scan %T into RuntimeHost", src)
	}
	return nil
}

func (h RuntimeHost) MarshalText() ([]byte, error) { return []byte(h.String()), nil }

func (h *RuntimeHost) UnmarshalText(b []byte) error {
	*h = ParseRuntimeHost(string(b))
	return nil
}

func (h RuntimeHost) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(h))
}

func (h *RuntimeHost) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*h = ParseRuntimeHost(s)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*h = RuntimeHost(n)
	return nil
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

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

// Non-standard request headers shared by clients (Flutter / Web / uniapp) and
// the Go server: lightweight identity plus the volatile live snapshot. The full
// profile is uploaded through UploadDeviceInfo instead.
//
// Wire names must stay stable, clients send them verbatim:
//   - Platform-Info: platform;channel;osVersion
//   - App-Info: appCode;appVersion
//   - Location (HeaderGeoLocation): lng;lat;area. The wire name is historical:
//     the standard Location header is a redirect header, see gox net/http.
//   - Device-Dynamic-Info: networkType;ramAvailMB;diskFreeB (missing segments
//     are left empty). It aggregates the volatile snapshot that previously
//     lived in the separate Network-Type / Ram-Avail / Disk-Free headers.
const (
	HeaderDeviceInfoMd5     = "Device-Info-Md5" // content-addressed device key (Device.StableMD5)
	HeaderPlatformInfo      = "Platform-Info"   // platform;channel;version (OS version)
	HeaderAppInfo           = "App-Info"        // appCode;appVersion
	HeaderGeoLocation       = "Location"        // lng;lat;area
	HeaderDeviceDynamicInfo = "Device-Dynamic-Info"
)

// NetworkType 网络类型。数值对齐 common.NetworkType。
type NetworkType int8

const (
	NetworkTypeUnspecified NetworkType = 0
	NetworkTypeWifi        NetworkType = 1
	NetworkTypeCellular    NetworkType = 2
	NetworkTypeEthernet    NetworkType = 3
	NetworkTypeVpn         NetworkType = 4
	NetworkTypeUnknown     NetworkType = 5
)

func (t NetworkType) String() string {
	switch t {
	case NetworkTypeWifi:
		return "wifi"
	case NetworkTypeCellular:
		return "cellular"
	case NetworkTypeEthernet:
		return "ethernet"
	case NetworkTypeVpn:
		return "vpn"
	case NetworkTypeUnknown:
		return "unknown"
	default:
		return ""
	}
}

// ParseNetworkType 解析 Device-Dynamic-Info / JSON 线格式（短名或数字）。
func ParseNetworkType(s string) NetworkType {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "0", "unspecified", "networktypeunspecified":
		return NetworkTypeUnspecified
	case "1", "wifi", "networktypewifi":
		return NetworkTypeWifi
	case "2", "cellular", "networktypecellular":
		return NetworkTypeCellular
	case "3", "ethernet", "networktypeethernet":
		return NetworkTypeEthernet
	case "4", "vpn", "networktypevpn":
		return NetworkTypeVpn
	case "5", "unknown", "networktypeunknown":
		return NetworkTypeUnknown
	default:
		if n, err := strconv.Atoi(s); err == nil {
			return NetworkType(n)
		}
		return NetworkTypeUnknown
	}
}

func (t NetworkType) Value() (driver.Value, error) { return int64(t), nil }

func (t *NetworkType) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*t = NetworkType(v)
	case int32:
		*t = NetworkType(v)
	case int:
		*t = NetworkType(v)
	case []byte:
		*t = ParseNetworkType(string(v))
	case string:
		*t = ParseNetworkType(v)
	case nil:
		*t = NetworkTypeUnspecified
	default:
		return fmt.Errorf("device: cannot scan %T into NetworkType", src)
	}
	return nil
}

func (t NetworkType) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

func (t *NetworkType) UnmarshalText(b []byte) error {
	*t = ParseNetworkType(string(b))
	return nil
}

func (t NetworkType) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(t))
}

func (t *NetworkType) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*t = ParseNetworkType(s)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*t = NetworkType(n)
	return nil
}

// Device 客户端上报的完整设备画像：稳定域（app/hardware/ident/os/host/network/web）
// 加 DeviceLite（身份 + 实时快照）。它只活在内存与上报链路里，不落库：稳定域按内容
// 寻址存进各自的 device_* 分表，主表 user_device 只留引用（见 UserDevice）。
// 实时快照经 DeviceLite.DeviceLiveInfo 带进来：不落库、不参与任何内容寻址 MD5。
type Device struct {
	UserID uint64 `json:"userId" gorm:"index"`

	DeviceLite // 身份（Md5/Platform/Channel/FormFactor/RuntimeHost）+ 实时快照（DeviceLiveInfo）

	App      DeviceAppInfo      `json:"app" gorm:"embedded;embeddedPrefix:app_"`
	Hardware DeviceHardwareInfo `json:"hardware" gorm:"embedded;embeddedPrefix:hw_"`
	ID       DeviceIDInfo       `json:"id" gorm:"embedded;embeddedPrefix:id_"`
	OS       DeviceOSInfo       `json:"os" gorm:"embedded;embeddedPrefix:os_"`
	Host     DeviceHostInfo     `json:"host" gorm:"embedded;embeddedPrefix:host_"`
	Network  DeviceNetworkInfo  `json:"network" gorm:"embedded;embeddedPrefix:net_"`
	Web      DeviceWebInfo      `json:"web" gorm:"embedded;embeddedPrefix:web_"`

	Ext map[string]string `json:"ext,omitempty" gorm:"serializer:json"`
}

// UserDevice 是 user_device 主表的行：归属某个用户，只保存稳定身份
// （md5/platform/channel）与各稳定域分表的内容寻址主键，不保存域明细与实时快照。
// ID 是自增代理主键；Md5 是客户端 Device-Info-Md5，业务唯一键。
type UserDevice struct {
	ID          uint64            `json:"id" gorm:"primaryKey"`
	UserID      uint64            `json:"userId" gorm:"index"`
	Md5         string            `json:"md5" gorm:"uniqueIndex;size:32"`
	Platform    DeviceOS          `json:"platform" gorm:"column:platform;type:smallint"`
	Channel     AppChannel        `json:"channel" gorm:"column:client_kind;type:smallint"`
	FormFactor  FormFactor        `json:"formFactor" gorm:"column:form_factor;type:smallint"`
	RuntimeHost RuntimeHost       `json:"runtimeHost" gorm:"column:runtime_host;type:smallint"`
	AppID       string            `json:"appId" gorm:"size:32;index"`
	HardwareID  string            `json:"hardwareId" gorm:"size:32;index"`
	IdentID     string            `json:"identId" gorm:"size:32;index"`
	OsID        string            `json:"osId" gorm:"size:32;index"`
	HostID      string            `json:"hostId" gorm:"size:32;index"`
	NetworkID   string            `json:"networkId" gorm:"size:32;index"`
	WebID       string            `json:"webId" gorm:"size:32;index"`
	Ext         map[string]string `json:"ext,omitempty" gorm:"serializer:json"`
	LastSeenAt  time.Time         `json:"lastSeenAt" gorm:"index"`
}

func (UserDevice) TableName() string { return "user_device" }

// DeviceAppInfo 应用 / 包信息。
type DeviceAppInfo struct {
	Code        string `json:"code" gorm:"size:128"`
	Name        string `json:"name" gorm:"size:128"`
	Version     string `json:"version" gorm:"size:64"`
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
	CPUVendor    string   `json:"cpuVendor" gorm:"size:64"`    // GenuineIntel / Apple / Qualcomm…
	CPUModel     string   `json:"cpuModel" gorm:"size:255"`    // 如 Apple M2 Pro / Intel Core i7-12700H
	CPUCores     int      `json:"cpuCores,omitempty"`          // 逻辑核；与 host.cpuCount 可同值
	GPUVendor    string   `json:"gpuVendor" gorm:"size:128"`   // Apple / Qualcomm / NVIDIA…
	GPUModel     string   `json:"gpuModel" gorm:"size:255"`    // 如 Apple M2 / Adreno 740
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
	MAC          string   `json:"mac" gorm:"size:64"` // relatively stable NIC/identifier MAC; volatile session MACs are dropped (modern OSes hide the real MAC from apps)
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
	OSSkin        string `json:"osSkin" gorm:"size:64"`        // OS 厂商衍生版 / 皮肤名（HyperOS/ColorOS/MIUI…）
	OSSkinVersion string `json:"osSkinVersion" gorm:"size:32"` // 皮肤自身版本，如 HyperOS 1.0
}

// DeviceHostInfo is the relatively stable host profile (locale / language / capacity).
// Volatile fields such as available RAM / free disk live in UserDevice.DeviceLiveInfo and must
// not be used as a device fingerprint.
type DeviceHostInfo struct {
	Locale       string `json:"locale" gorm:"size:64"`
	Language     string `json:"language" gorm:"size:32"`
	Languages    string `json:"languages" gorm:"size:255"`
	Timezone     string `json:"timezone" gorm:"size:64"`
	Hostname     string `json:"hostname" gorm:"size:255"`
	CPUCount     int    `json:"cpuCount,omitempty"`
	CPUFrequency int64  `json:"cpuFrequencyHz,omitempty"` // 标称/基准频率；DVFS 瞬时值勿填这里
	RamMB        int64  `json:"ramMb,omitempty"`          // 物理内存总量
	DiskTotalB   int64  `json:"diskTotalBytes,omitempty"` // 磁盘总量
}

// DeviceLiveInfo is a volatile per-request snapshot (RAM/disk/network/geo).
// It changes with process/OS load, sessions and connections, so it must not be
// used as a device fingerprint and is not persisted.
// Volatile MACs (wifiMac/bluetoothMac) are dropped: modern OSes hide the real
// MAC from apps and hand out randomized values with no stable identity.
type DeviceLiveInfo struct {
	IP           net.IP      `json:"ip,omitempty" gorm:"size:64"`
	NetworkType  NetworkType `json:"networkType,omitempty" gorm:"type:smallint"` // wifi|cellular|ethernet|vpn|unknown
	Lng          float64     `json:"lng,omitempty" gorm:"type:numeric(10,6)"`
	Lat          float64     `json:"lat,omitempty" gorm:"type:numeric(10,6)"`
	Area         string      `json:"area,omitempty" gorm:"size:255"`
	UserAgent    string      `json:"userAgent,omitempty" gorm:"size:512"`
	RamAvailMB   int64       `json:"ramAvailMB,omitempty" gorm:"type:bigint"`
	DiskFreeB    int64       `json:"diskFreeB,omitempty" gorm:"type:bigint"`
	ScreenWidth  int         `json:"screenWidth,omitempty"`
	ScreenHeight int         `json:"screenHeight,omitempty"`
	PixelRatio   float64     `json:"pixelRatio,omitempty"`
}

// DeviceNetworkInfo is the relatively stable carrier / SIM profile.
// Volatile session-side fields (IP/network-type/geo) live in UserDevice.DeviceLiveInfo.
type DeviceNetworkInfo struct {
	Carrier string `json:"carrier" gorm:"size:64"`
	ICCID   string `json:"iccid" gorm:"size:32"`
	IMSI    string `json:"imsi" gorm:"size:32"`
}

// DeviceWebInfo 浏览器 / WebView。
type DeviceWebInfo struct {
	UserAgent      string   `json:"userAgent" gorm:"size:512"`
	Browser        string   `json:"browser" gorm:"size:64"`
	BrowserVersion string   `json:"browserVersion" gorm:"size:64"`
	Engine         string   `json:"engine" gorm:"size:64"`
	EngineVersion  string   `json:"engineVersion" gorm:"size:64"`
	Vendor         string   `json:"vendor" gorm:"size:128"`
	MaxTouchPts    int      `json:"maxTouchPoints,omitempty"`
	ColorDepth     int      `json:"colorDepth,omitempty"`
	DeviceMemoryG  float64  `json:"deviceMemoryGb,omitempty"`
	CookieEnabled  TriState `json:"cookieEnabled,omitempty" gorm:"type:smallint"`
	DoNotTrack     TriState `json:"doNotTrack,omitempty" gorm:"type:smallint"`
}

// DeviceLite is a lightweight per-request device snapshot (isomorphic to
// user.AccessDevice). It is built from Platform-Info / App-Info / Location /
// Device-Info-Md5 / UA / XFF / Device-Dynamic-Info headers without reconstructing
// a full UserDevice. The volatile fields are embedded via DeviceLiveInfo.
type DeviceLite struct {
	Md5            string      `json:"md5" gorm:"uniqueIndex;size:32"`
	Platform       DeviceOS    `json:"platform" gorm:"type:smallint"`
	Channel        AppChannel  `json:"channel" gorm:"type:smallint"`
	FormFactor     FormFactor  `json:"formFactor" gorm:"type:smallint"`
	RuntimeHost    RuntimeHost `json:"runtimeHost" gorm:"type:smallint"`
	Version        string      `json:"version" gorm:"size:64"` // 系统版本（OS.Version）
	AppCode        string      `json:"appCode" gorm:"size:255"`
	AppVersion     string      `json:"appVersion" gorm:"size:255"`
	DeviceLiveInfo             // volatile per-request snapshot
}

// Empty 是否全空。
func (l DeviceLite) Empty() bool {
	return l.Md5 == "" &&
		l.Platform == DeviceOSUnspecified && l.Channel == AppChannelUnspecified &&
		l.Version == "" &&
		l.AppCode == "" && l.AppVersion == "" &&
		l.IP == nil && l.Area == "" &&
		l.Lng == 0 && l.Lat == 0 &&
		l.UserAgent == ""
}

// Lite projects the device into the lightweight DeviceLite snapshot.
// The embedded DeviceLite is already the identity + live snapshot, so only the
// fields that have to be derived from the stable domains are filled in when empty.
func (d *Device) Lite() DeviceLite {
	if d == nil {
		return DeviceLite{}
	}
	d.Normalize()
	lite := d.DeviceLite
	if lite.Md5 == "" {
		lite.Md5 = d.PrimaryDeviceNo()
	}
	if lite.Version == "" {
		lite.Version = d.OS.Version
	}
	if lite.AppCode == "" {
		lite.AppCode = d.App.Code
	}
	if lite.AppVersion == "" {
		lite.AppVersion = d.App.Version
	}
	return lite
}

// DisplayName 人可读设备名。
func (d *Device) DisplayName() string {
	if d == nil {
		return ""
	}
	h := d.Hardware
	return firstNonEmpty(h.ModelName, h.Model, h.DeviceName, h.Brand, h.Product)
}

// OSDisplay 人可读系统版本。
func (d *Device) OSDisplay() string {
	if d == nil {
		return ""
	}
	return strings.TrimSpace(firstNonEmpty(d.OS.Name, d.Platform.String()) + " " + d.OS.Version)
}

// PrimaryDeviceNo 优先业务 DID，再按常见标识回退。
func (d *Device) PrimaryDeviceNo() string {
	if d == nil {
		return ""
	}
	id := d.ID
	return firstNonEmpty(
		id.DID, id.IDFV, id.AndroidID, id.OAID, id.GAID, id.AAID,
		id.IDFA, id.GUID, id.UUID, id.UDID, id.OpenUDID, id.IMEI, id.MAC,
	)
}

// Normalize 补全 platform 等可推导字段。channel / formFactor / runtimeHost 由客户端显式上报，不推断。
func (d *Device) Normalize() {
	if d == nil {
		return
	}
	if d.Platform == DeviceOSUnspecified {
		d.Platform = inferPlatform(d)
	}
	if d.OS.OSSkin != "" {
		d.OS.OSSkin = strings.TrimSpace(d.OS.OSSkin)
	}
	if d.App.Version == "" && d.App.Build != "" {
		d.App.Version = d.App.Build
	}
	if d.App.PackageName == "" && d.App.BundleID != "" {
		d.App.PackageName = d.App.BundleID
	}
	if d.ID.GAID == "" && d.ID.AAID != "" {
		d.ID.GAID = d.ID.AAID
	}
	if d.ID.AAID == "" && d.ID.GAID != "" {
		d.ID.AAID = d.ID.GAID
	}
	if d.Web.UserAgent == "" && d.Web.Browser != "" {
		d.Web.UserAgent = strings.TrimSpace(d.Web.Browser + "/" + d.Web.BrowserVersion)
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

func inferPlatform(d *Device) DeviceOS {
	s := strings.ToLower(strings.TrimSpace(d.OS.Name + " " + d.OS.OSSkin + " " + d.OS.Edition + " " + d.Platform.String()))
	switch {
	case strings.Contains(s, "android"):
		return DeviceOSAndroid
	// 厂商 OS 衍生版（均为 Android 衍生），只上报皮肤名时也能归到 Android。
	case strings.Contains(s, "hyperos"), strings.Contains(s, "coloros"), strings.Contains(s, "miui"),
		strings.Contains(s, "one ui"), strings.Contains(s, "originos"), strings.Contains(s, "flyme"),
		strings.Contains(s, "emui"), strings.Contains(s, "magicos"), strings.Contains(s, "funtouchos"):
		return DeviceOSAndroid
	case strings.Contains(s, "ios"), strings.Contains(s, "iphone"), strings.Contains(s, "ipad"):
		return DeviceOSIos
	case strings.Contains(s, "macos"), strings.Contains(s, "darwin"), strings.Contains(s, "mac os"):
		return DeviceOSMacos
	case strings.Contains(s, "windows"):
		return DeviceOSWindows
	case strings.Contains(s, "linux"):
		return DeviceOSLinux
	case strings.Contains(s, "harmonyos"), strings.Contains(s, "harmony"), strings.Contains(s, "hongmeng"):
		return DeviceOSHarmonyOS
	case d.ID.IDFV != "" || d.ID.IDFA != "":
		return DeviceOSIos
	case d.ID.AndroidID != "" || d.ID.OAID != "" || d.ID.GAID != "" || d.ID.IMEI != "":
		return DeviceOSAndroid
	default:
		return DeviceOSUnspecified
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

// LiteFromHeader 从请求头组装 DeviceLite（= AccessDevice）。
// Platform-Info = platform;channel;version
// App-Info = appCode;appVersion
// Location = lng;lat;area
// 另含 Device-Info-Md5 / User-Agent / X-Forwarded-For。
func LiteFromHeader(header http.Header) DeviceLite {
	var lite DeviceLite
	if header == nil {
		return lite
	}
	platParts := splitSemiHeader(header.Get(HeaderPlatformInfo))
	if len(platParts) > 0 {
		lite.Platform = ParseDeviceOS(platParts[0])
	}
	if len(platParts) > 1 {
		lite.Channel = ParseAppChannel(platParts[1])
	}
	if len(platParts) > 2 {
		lite.Version = platParts[2]
	}
	appParts := splitSemiHeader(header.Get(HeaderAppInfo))
	if len(appParts) > 0 {
		lite.AppCode = appParts[0]
	}
	if len(appParts) > 1 {
		lite.AppVersion = appParts[1]
	}
	if loc := header.Get(HeaderGeoLocation); loc != "" {
		lite.Lng, lite.Lat, lite.Area = parseLocationHeader(loc)
	}
	if xff := header.Get(httpx.HeaderXForwardedFor); xff != "" {
		if ip := net.ParseIP(firstForwardedIP(xff)); ip != nil {
			lite.IP = ip
		}
	}
	lite.UserAgent = header.Get(httpx.HeaderUserAgent)
	// 与 Platform-Info / App-Info 同风格：分号分隔、只带值，
	// 顺序为 networkType;ramAvailMB;diskFreeB（缺位用空段占位）。
	if parts := splitSemiHeader(header.Get(HeaderDeviceDynamicInfo)); len(parts) > 0 {
		lite.NetworkType = ParseNetworkType(parts[0])
		if len(parts) > 1 {
			if n, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				lite.RamAvailMB = n
			}
		}
		if len(parts) > 2 {
			if n, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
				lite.DiskFreeB = n
			}
		}
	}
	lite.Md5 = strings.TrimSpace(header.Get(HeaderDeviceInfoMd5))
	return lite
}

func parseLocationHeader(raw string) (lng, lat float64, area string) {
	parts := splitSemiHeader(raw)
	if len(parts) > 0 {
		lng, _ = strconv.ParseFloat(parts[0], 64)
	}
	if len(parts) > 1 {
		lat, _ = strconv.ParseFloat(parts[1], 64)
	}
	if len(parts) > 2 {
		area = parts[2]
	}
	return lng, lat, area
}

func splitSemiHeader(raw string) []string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	if unescaped, err := url.QueryUnescape(s); err == nil {
		s = strings.TrimSpace(unescaped)
	}
	parts := strings.Split(s, ";")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// LivePresent reports whether a volatile snapshot (RAM/disk/network-type/IP/geo/UA) is present.
func LivePresent(live DeviceLiveInfo) bool {
	return live.RamAvailMB != 0 || live.DiskFreeB != 0 ||
		live.NetworkType != NetworkTypeUnspecified ||
		live.IP != nil || live.Area != "" || live.Lng != 0 || live.Lat != 0 ||
		live.UserAgent != ""
}

// HasLive reports whether this device carries a volatile snapshot.
func (d *Device) HasLive() bool {
	if d == nil {
		return false
	}
	return LivePresent(d.DeviceLiveInfo) || d.Web.UserAgent != ""
}

// Empty 是否缺少稳定设备画像（live 头不算身份）。
func (d *Device) Empty() bool {
	if d == nil {
		return true
	}
	platEmpty := d.Platform == DeviceOSUnspecified || d.Platform == DeviceOSUnknown
	return platEmpty &&
		d.DisplayName() == "" && d.PrimaryDeviceNo() == "" &&
		d.App.Code == "" && d.App.Version == "" && d.Web.Browser == "" &&
		d.OS.Name == "" && d.OS.Version == "" && len(d.Ext) == 0
}

func DeviceFromJSON(raw string) *Device {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	if !strings.HasPrefix(s, "{") {
		return nil
	}
	info := new(Device)
	if err := json.Unmarshal([]byte(s), info); err != nil {
		return nil
	}
	return info
}

func firstForwardedIP(xff string) string {
	if i := strings.IndexByte(xff, ','); i >= 0 {
		return strings.TrimSpace(xff[:i])
	}
	return strings.TrimSpace(xff)
}
