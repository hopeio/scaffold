package device

import (
	"testing"
	"time"
)

func sp(s string) *string {
	return &s
}
func ip(i int) *int            { return &i }
func ptrInt64(v int64) *int64  { return &v }
func itp(t TriState) *TriState { return &t }

func TestCanonicalIdentityPrefersPlatformIDs(t *testing.T) {
	cases := []struct {
		name     string
		p        *IdentPatch
		wantKey  string
		wantKind KeyKind
	}{
		{"nil domain has no identity", nil, "", KeyKindUnspecified},
		{"empty domain has no identity", &IdentPatch{}, "", KeyKindUnspecified},
		{"DID only", &IdentPatch{DID: sp("did-1")}, "did-1", KeyKindDID},
		// A DID dies on reinstall while a platform id outlives app upgrades, so when
		// both are present the platform id must win.
		{"IDFV wins over DID", &IdentPatch{DID: sp("did-1"), IDFV: sp("idfv-9")}, "idfv-9", KeyKindIDFV},
		{"AndroidID wins over the resettable OAID", &IdentPatch{OAID: sp("oaid-1"), AndroidID: sp("aid-1")}, "aid-1", KeyKindAndroidID},
		{"OAID wins over GAID", &IdentPatch{GAID: sp("gaid-1"), OAID: sp("oaid-1")}, "oaid-1", KeyKindOAID},
		// An empty string means the user withdrew that id, so it is not an identity:
		// resolution has to fall through to the next available one.
		{"withdrawn IDFV falls back to DID", &IdentPatch{IDFV: sp(""), DID: sp("did-1")}, "did-1", KeyKindDID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k, kind := CanonicalIdentity(c.p)
			if k != c.wantKey || kind != c.wantKind {
				t.Fatalf("got (%q,%v) want (%q,%v)", k, kind, c.wantKey, c.wantKind)
			}
		})
	}
}

// The two overwrite classes are the core of this change: one failed read of a spec
// field must not wipe a real value, while a withdrawn identifier (present, empty)
// must overwrite, or the store keeps an id the user already revoked.
func TestDeviceColumnsOverwriteClasses(t *testing.T) {
	u := &Upload{
		Platform: DeviceOSAndroid,
		Hardware: &HardwarePatch{Model: sp("Pixel 9"), Board: sp(""), CPUCores: ip(0)},
		Ident:    &IdentPatch{DID: sp("d"), IDFATracking: itp(TriFalse)},
		Web:      &WebPatch{MaxTouchPoints: ip(0), ColorDepth: ip(0)},
		Host:     &HostPatch{Timezone: sp("Asia/Shanghai"), RamMb: ptrInt64(0)},
	}
	m := u.DeviceColumns()

	if got := m["model"]; got != "Pixel 9" {
		t.Errorf("non-zero spec value must be written, got %v", got)
	}
	if _, ok := m["board"]; ok {
		t.Error("empty board (unreadable / not on this platform) must not overwrite")
	}
	if _, ok := m["cpu_cores"]; ok {
		t.Error("0 cores cannot be real, so a 0 read must not overwrite")
	}
	if _, ok := m["ram_mb"]; ok {
		t.Error("0 MB RAM likewise must not overwrite")
	}
	if got := m["timezone"]; got != "Asia/Shanghai" {
		t.Errorf("timezone should be written, got %v", got)
	}
	// maxTouchPoints is the deliberate exception: a non-touch screen really is 0.
	if got, ok := m["max_touch_pts"]; !ok || got != 0 {
		t.Errorf("maxTouchPoints=0 is a real value and must be written, got %v ok=%v", got, ok)
	}
	if _, ok := m["color_depth"]; ok {
		t.Error("colorDepth=0 reads as a failed read and must not overwrite")
	}
	if got := m["idfa_tracking"]; got != int8(TriFalse) {
		t.Errorf("an explicit TriState false must be written, got %v", got)
	}
	if v := m["platform"]; v != int16(DeviceOSAndroid) {
		t.Errorf("enums are always written, got %v", v)
	}
	// Versions describe an app on the device and must never leak into device facts.
	for _, k := range []string{"app_version", "app_build", "version", "build", "service_name"} {
		if _, ok := m[k]; ok {
			t.Errorf("device columns must not contain the app field %s", k)
		}
	}
}

func TestTriStateUnsetIsNotReported(t *testing.T) {
	m := map[string]any{}
	putTri(m, "is_physical", itp(TriUnset))
	if len(m) != 0 {
		t.Fatalf("Unset already encodes unknown for this type; writing it again gives two ways to say the same thing: %v", m)
	}
	putTri(m, "is_physical", itp(TriTrue))
	if m["is_physical"] != int8(TriTrue) {
		t.Errorf("an explicit true must be written, got %v", m["is_physical"])
	}
}

func TestInstallColumnsNeedsDomainAndCode(t *testing.T) {
	var u Upload
	if _, ok := u.InstallColumns(); ok {
		t.Fatal("no app domain reported means no install write at all")
	}
	u.App = &AppPatch{Name: sp("rfv")} // domain present but no code
	m, ok := u.InstallColumns()
	if !ok {
		t.Fatal("a reported app domain returns ok; the caller decides on missing code")
	}
	if _, has := m["app_code"]; has {
		t.Error("an unreported code must not be written as an empty code")
	}
	u.App.Code = sp("rfv")
	m, _ = u.InstallColumns()
	if m["app_code"] != "rfv" {
		t.Errorf("code should be written, got %v", m["app_code"])
	}
}

func TestDiffColumnsOnlyRealChanges(t *testing.T) {
	old := map[string]any{"os_version": "18.1", "ram_mb": int64(16384), "language": "zh"}
	want := map[string]any{"os_version": "18.2", "ram_mb": int64(16384), "locale": "zh_CN"}
	got := diffColumns(old, want)
	if len(got) != 2 {
		t.Fatalf("only os_version and the new locale should appear, got %+v", got)
	}
	var seenVer, seenLocale bool
	for _, c := range got {
		switch c.FieldName {
		case "os_version":
			seenVer = c.OldValue == "18.1" && c.NewValue == "18.2"
		case "locale":
			seenLocale = c.OldValue == "" && c.NewValue == "zh_CN"
		case "ram_mb":
			t.Errorf("equal values must not be logged (int64 vs int denote the same number): %+v", c)
		}
	}
	if !seenVer || !seenLocale {
		t.Fatalf("change contents wrong: %+v", got)
	}
}

func TestExtJSONEmptyNotWritten(t *testing.T) {
	if _, ok := extJSON(nil); ok {
		t.Error("an empty ext map must not be written, or it erases stored extension keys")
	}
	s, ok := extJSON(map[string]string{"b": "2", "a": "1"})
	if !ok || s != `{"a":"1","b":"2"}` {
		t.Errorf("ext must serialise key-sorted, otherwise equal content yields different values: %q", s)
	}
}

func TestApplyRefusesUploadWithoutIdentity(t *testing.T) {
	// Identity is validated before anything else: writing with an empty key would make
	// every "identity unknown" upload collide into one shared device row.
	if _, err := Apply(nil, &Upload{Ident: &IdentPatch{}}, time.Time{}); err != ErrNoIdentity {
		t.Fatalf("want ErrNoIdentity, got %v", err)
	}
}
