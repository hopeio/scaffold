// Package objectthumb turns an original image stored in an object store into a small
// WebP rendition for list/grid surfaces, so scrolling never downloads originals.
//
// WebP (lossy, VP8) instead of JPEG: same visual quality at roughly half the bytes, and
// it keeps alpha - logos and stickers stop getting a white box. Encoding goes through
// libwebp compiled to WASM (gen2brain/webp), so it runs with CGO_ENABLED=0 and needs no
// system library; the server image is distroless/static and cannot load libwebp.so.
//
// It deliberately owns only the image work plus a two-method Store interface: scanning
// rows and persisting the resulting key stays with the calling domain, which keeps this
// package free of any database dependency and lets content/chat share one implementation
// without importing each other's DAO.
package objectthumb

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"strings"

	"github.com/gen2brain/webp"
)

const (
	// DefaultMaxEdge is the long edge of a rendition. Grid cells are ~300 logical pixels,
	// so 480 covers a 2x screen.
	DefaultMaxEdge = 480
	// DefaultQuality is the WebP quality used for renditions. WebP q80 is visually
	// comparable to JPEG q85 at a fraction of the size.
	DefaultQuality = 80
	// MaxSourceBytes caps the bytes read for one source image; bigger files are reported
	// as unavailable instead of being pulled into memory.
	MaxSourceBytes = 20 << 20
	// ContentType is what callers must write with the rendition object. It also decides
	// how clients decode it, so it is part of the contract rather than an implementation
	// detail of the store adapter.
	ContentType = "image/webp"
	// UnavailableKey is the sentinel callers persist when a source cannot produce a
	// rendition (unsupported format, undecodable bytes, or already small enough). Read
	// paths must treat it as "no thumb" and fall back to the original; persisting it is
	// what keeps a broken image out of the retry loop.
	UnavailableKey = "-"
)

// Supported reports whether the standard library can decode this MIME type. webp/avif/
// heic are intentionally excluded: decoding them would need cgo or a third-party
// decoder, and a wrong guess produces a broken file rather than a missing one.
func Supported(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/png":
		return true
	default:
		return false
	}
}

// Store is the minimal object-store surface Generate needs. Put receives the content
// type to persist with the object (see ContentType).
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, data []byte, contentType string) error
}

// Result describes one generation attempt.
type Result struct {
	// ThumbKey is the stored rendition key; empty when Unavailable.
	ThumbKey string
	// ContentType is the MIME type persisted with ThumbKey.
	ContentType string
	// Unavailable means "do not retry": persist UnavailableKey and fall back to the
	// original on read.
	Unavailable bool
	// Reason is for logs only.
	Reason string
}

// Generate produces a rendition of srcKey and stores it. Storage failures come back as
// err (the caller should retry next round); image problems come back as
// Result{Unavailable: true} (the caller should persist the sentinel).
func Generate(ctx context.Context, store Store, srcKey, mime string, maxEdge int) (Result, error) {
	if store == nil || strings.TrimSpace(srcKey) == "" {
		return Result{Unavailable: true, Reason: "empty source"}, nil
	}
	if !Supported(mime) {
		return Result{Unavailable: true, Reason: "unsupported mime: " + mime}, nil
	}
	if maxEdge <= 0 {
		maxEdge = DefaultMaxEdge
	}
	raw, err := store.Get(ctx, srcKey)
	if err != nil {
		return Result{}, err
	}
	if len(raw) > MaxSourceBytes {
		return Result{Unavailable: true, Reason: "source too large"}, nil
	}
	src, err := decode(raw, mime)
	if err != nil {
		return Result{Unavailable: true, Reason: "decode failed"}, nil
	}
	b := src.Bounds()
	if longestEdge(b.Dx(), b.Dy()) <= maxEdge {
		return Result{Unavailable: true, Reason: "already small"}, nil
	}
	out, err := EncodeWebP(src, maxEdge, DefaultQuality)
	if err != nil {
		return Result{Unavailable: true, Reason: "encode failed"}, nil
	}
	key := ObjectKey(srcKey)
	if err := store.Put(ctx, key, out, ContentType); err != nil {
		return Result{}, err
	}
	return Result{ThumbKey: key, ContentType: ContentType}, nil
}

// ObjectKey derives the rendition key from the source key.
//
// The source key can approach the column width (255), so a prefix would overflow; a
// truncated SHA-256 is fixed-length and collision-free in practice. The "thumb/" prefix
// keeps renditions in their own namespace inside the bucket.
func ObjectKey(srcKey string) string {
	sum := sha256.Sum256([]byte(srcKey))
	return "thumb/" + hex.EncodeToString(sum[:16]) + ".webp"
}

// TargetSize scales (w, h) down so the long edge equals maxEdge at most. Both sides are
// at least 1px; non-positive input returns (0, 0).
func TargetSize(w, h, maxEdge int) (int, int) {
	if w <= 0 || h <= 0 || maxEdge <= 0 {
		return 0, 0
	}
	longest := longestEdge(w, h)
	if longest <= maxEdge {
		return w, h
	}
	scale := float64(maxEdge) / float64(longest)
	tw := int(math.Round(float64(w) * scale))
	th := int(math.Round(float64(h) * scale))
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}
	return tw, th
}

// EncodeWebP downscales the image and encodes a lossy WebP.
//
// Alpha is preserved: WebP carries it, so transparent PNGs (logos, stickers) no longer
// get a white rectangle behind them. The source is copied into RGBA with draw.Src for
// that reason - draw.Over would blend against an empty (black) destination.
func EncodeWebP(src image.Image, maxEdge, quality int) ([]byte, error) {
	sb := src.Bounds()
	tw, th := TargetSize(sb.Dx(), sb.Dy(), maxEdge)
	if tw == 0 || th == 0 {
		return nil, errors.New("invalid target size")
	}
	if quality <= 0 {
		quality = DefaultQuality
	}
	rgba := image.NewRGBA(image.Rect(0, 0, sb.Dx(), sb.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, sb.Min, draw.Src)
	scaled := resizeBilinear(rgba, tw, th)
	var buf bytes.Buffer
	if err := webp.Encode(&buf, scaled, webp.Options{Quality: quality, Method: 4}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decode(raw []byte, mime string) (image.Image, error) {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg":
		return jpeg.Decode(bytes.NewReader(raw))
	case "image/png":
		return png.Decode(bytes.NewReader(raw))
	default:
		return nil, image.ErrFormat
	}
}

// resizeBilinear scales an RGBA image, alpha included. The standard library has no
// scaler; this runs only in background workers, so clarity beats micro-optimisation.
func resizeBilinear(src *image.RGBA, tw, th int) *image.RGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	if sw == 0 || sh == 0 || tw == 0 || th == 0 {
		return dst
	}
	scaleX := float64(sw) / float64(tw)
	scaleY := float64(sh) / float64(th)
	for y := 0; y < th; y++ {
		fy := (float64(y)+0.5)*scaleY - 0.5
		y0 := int(math.Floor(fy))
		wy := fy - float64(y0)
		if y0 < 0 {
			y0, wy = 0, 0
		}
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		for x := 0; x < tw; x++ {
			fx := (float64(x)+0.5)*scaleX - 0.5
			x0 := int(math.Floor(fx))
			wx := fx - float64(x0)
			if x0 < 0 {
				x0, wx = 0, 0
			}
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			c00 := src.RGBAAt(sb.Min.X+x0, sb.Min.Y+y0)
			c10 := src.RGBAAt(sb.Min.X+x1, sb.Min.Y+y0)
			c01 := src.RGBAAt(sb.Min.X+x0, sb.Min.Y+y1)
			c11 := src.RGBAAt(sb.Min.X+x1, sb.Min.Y+y1)
			mix := func(v00, v10, v01, v11 uint8) uint8 {
				top := float64(v00)*(1-wx) + float64(v10)*wx
				bottom := float64(v01)*(1-wx) + float64(v11)*wx
				return uint8(math.Round(top*(1-wy) + bottom*wy))
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: mix(c00.R, c10.R, c01.R, c11.R),
				G: mix(c00.G, c10.G, c01.G, c11.G),
				B: mix(c00.B, c10.B, c01.B, c11.B),
				A: mix(c00.A, c10.A, c01.A, c11.A),
			})
		}
	}
	return dst
}

func longestEdge(w, h int) int {
	if w > h {
		return w
	}
	return h
}
