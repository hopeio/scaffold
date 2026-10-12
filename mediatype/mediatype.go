// Package mediatype classifies stored objects by file extension or MIME type.
//
// Pure logic: no I/O, no global dependency, so every domain (moments, chat,
// admin uploads) shares one whitelist instead of drifting copies.
package mediatype

import (
	"strings"
)

// Kind is the media class of an object.
type Kind uint8

const (
	// KindUnknown means the extension is in no whitelist.
	KindUnknown Kind = iota
	KindImage
	KindVideo
	KindVoice
	// KindFile is the catch-all for documents and archives.
	KindFile
)

// String renders the kind for logs and metrics labels.
func (k Kind) String() string {
	switch k {
	case KindImage:
		return "image"
	case KindVideo:
		return "video"
	case KindVoice:
		return "voice"
	case KindFile:
		return "file"
	default:
		return "unknown"
	}
}

// exts holds the per-kind extension whitelist (lower-case, no dot).
//
// The video set is deliberately narrow: only containers browsers and mobile
// players handle natively. Broadening it is a product decision, not a bugfix.
var exts = map[Kind]map[string]struct{}{
	KindImage: {
		"jpg": {}, "jpeg": {}, "png": {}, "gif": {}, "webp": {},
		"bmp": {}, "heic": {}, "heif": {}, "avif": {},
	},
	KindVideo: {
		"mp4": {}, "mov": {}, "m4v": {}, "webm": {},
	},
	KindVoice: {
		"m4a": {}, "aac": {}, "mp3": {}, "wav": {}, "ogg": {}, "opus": {}, "amr": {},
	},
}

// mimePrefixes maps a MIME prefix to a kind; KindFile has none because its
// MIME space is open-ended (application/*, text/*, multipart/*).
var mimePrefixes = []struct {
	prefix string
	kind   Kind
}{
	{"image/", KindImage},
	{"video/", KindVideo},
	{"audio/", KindVoice},
}

// MaxExtLen caps the extension length; longer means the "extension" is really
// part of the file name, so it carries no classification value.
const MaxExtLen = 16

// Ext returns the lower-cased extension without the dot, or "" when the name
// has none or the extension is implausibly long.
func Ext(nameOrKey string) string {
	i := strings.LastIndexByte(nameOrKey, '.')
	if i < 0 || i == len(nameOrKey)-1 {
		return ""
	}
	ext := strings.ToLower(nameOrKey[i+1:])
	if len(ext) > MaxExtLen || strings.ContainsAny(ext, `/\?#`) {
		return ""
	}
	return ext
}

// KindOf classifies by extension; KindUnknown when not whitelisted.
func KindOf(nameOrKey string) Kind {
	ext := Ext(nameOrKey)
	if ext == "" {
		return KindUnknown
	}
	for kind, set := range exts {
		if _, ok := set[ext]; ok {
			return kind
		}
	}
	return KindUnknown
}

// KindOfMime classifies by MIME type; KindUnknown when the prefix is not
// image/, video/ or audio/.
func KindOfMime(mime string) Kind {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if mime == "" {
		return KindUnknown
	}
	if i := strings.IndexByte(mime, ';'); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	for _, m := range mimePrefixes {
		if strings.HasPrefix(mime, m.prefix) {
			return m.kind
		}
	}
	return KindUnknown
}

// Allowed reports whether nameOrKey may be used as the given kind.
//
// KindFile has no extension whitelist: users legitimately share documents and
// archives of any type, so the caller enforces size and count limits instead.
// An empty name is rejected for every kind, including KindFile.
func Allowed(kind Kind, nameOrKey string) bool {
	nameOrKey = strings.TrimSpace(nameOrKey)
	if nameOrKey == "" {
		return false
	}
	if kind == KindFile {
		return true
	}
	set := exts[kind]
	if set == nil {
		return false
	}
	_, ok := set[Ext(nameOrKey)]
	return ok
}
