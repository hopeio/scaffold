package mediatype

import "testing"

func TestExt(t *testing.T) {
	cases := []struct{ in, want string }{
		{"image/2026/abc_123.JPG", "jpg"},
		{"voice.m4a", "m4a"},
		{"noext", ""},
		{"trailing.", ""},
		{"a.verylongextensionname", ""},
		{"dir.d/name", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := Ext(c.in); got != c.want {
			t.Errorf("Ext(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKindOf(t *testing.T) {
	cases := []struct {
		in   string
		want Kind
	}{
		{"a/photo.PNG", KindImage},
		{"a/clip.mp4", KindVideo},
		{"a/rec.opus", KindVoice},
		{"a/doc.pdf", KindUnknown},
		{"a/noext", KindUnknown},
	}
	for _, c := range cases {
		if got := KindOf(c.in); got != c.want {
			t.Errorf("KindOf(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestKindOfMime(t *testing.T) {
	cases := []struct {
		in   string
		want Kind
	}{
		{"image/jpeg", KindImage},
		{"VIDEO/MP4", KindVideo},
		{"audio/mpeg; codecs=mp3", KindVoice},
		{"application/pdf", KindUnknown},
		{"", KindUnknown},
	}
	for _, c := range cases {
		if got := KindOfMime(c.in); got != c.want {
			t.Errorf("KindOfMime(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestAllowed(t *testing.T) {
	if !Allowed(KindImage, "a/photo.webp") {
		t.Error("webp should be an allowed image")
	}
	if Allowed(KindImage, "a/clip.mp4") {
		t.Error("mp4 must not pass as image")
	}
	if !Allowed(KindVideo, "a/clip.mov") {
		t.Error("mov should be an allowed video")
	}
	if Allowed(KindVideo, "a/clip.mkv") {
		t.Error("mkv is outside the playable video whitelist")
	}
	if !Allowed(KindVoice, "a/rec.amr") {
		t.Error("amr should be an allowed voice")
	}
	// Documents have no extension whitelist, but a blank name is never allowed.
	if !Allowed(KindFile, "a/archive.tar.gz") {
		t.Error("any document should pass as file")
	}
	if Allowed(KindFile, "   ") {
		t.Error("blank name must not pass")
	}
	if Allowed(KindUnknown, "a/photo.png") {
		t.Error("KindUnknown has no whitelist")
	}
}

func TestKindString(t *testing.T) {
	if KindVoice.String() != "voice" || KindUnknown.String() != "unknown" {
		t.Errorf("unexpected kind strings: %v %v", KindVoice, KindUnknown)
	}
}
