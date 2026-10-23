package objectthumb

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/gen2brain/webp"
)

func TestTargetSize(t *testing.T) {
	cases := []struct{ w, h, max, tw, th int }{
		{4000, 3000, 480, 480, 360},
		{3000, 4000, 480, 360, 480},
		{480, 100, 480, 480, 100},
		{100, 50, 480, 100, 50},
		{1000, 1, 480, 480, 1},
		{0, 100, 480, 0, 0},
		{100, 100, 0, 0, 0},
	}
	for _, c := range cases {
		tw, th := TargetSize(c.w, c.h, c.max)
		if tw != c.tw || th != c.th {
			t.Errorf("TargetSize(%d,%d,%d) = %d,%d want %d,%d", c.w, c.h, c.max, tw, th, c.tw, c.th)
		}
	}
}

func TestObjectKey(t *testing.T) {
	key := "image/2026/10/abc_123.jpg"
	got := ObjectKey(key)
	// The rendition is WebP regardless of the source extension.
	if !strings.HasPrefix(got, "thumb/") || !strings.HasSuffix(got, ".webp") {
		t.Fatalf("unexpected key: %s", got)
	}
	// The source key can approach the 255-char column width; the rendition key must not.
	if len(got) != len("thumb/")+32+len(".webp") {
		t.Fatalf("length %d: %s", len(got), got)
	}
	if ObjectKey(key) != got {
		t.Fatal("must be stable")
	}
	if ObjectKey(key+".bak") == got {
		t.Fatal("different sources must differ")
	}
}

// WebP carries alpha, so a transparent PNG must decode back transparent (JPEG used to
// force a white box there).
func TestEncodeWebPKeepsAlpha(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for y := 0; y < 400; y++ {
		for x := 400; x < 800; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out, err := EncodeWebP(src, 480, DefaultQuality)
	if err != nil {
		t.Fatal(err)
	}
	img, err := webp.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 480 || b.Dy() != 240 {
		t.Fatalf("size %dx%d", b.Dx(), b.Dy())
	}
	_, _, _, a := img.At(10, 10).RGBA()
	if a != 0 {
		t.Fatalf("transparent corner became opaque: a=%d", a)
	}
	r, _, _, a2 := img.At(470, 10).RGBA()
	if a2 == 0 || r == 0 {
		t.Fatalf("opaque region lost: a=%d r=%d", a2, r)
	}
}

type memStore struct {
	objects map[string][]byte
	puts    []string
	types   map[string]string
	getErr  error
	putErr  error
}

func (m *memStore) Get(_ context.Context, key string) ([]byte, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	b, ok := m.objects[key]
	if !ok {
		return nil, errors.New("missing")
	}
	return b, nil
}

func (m *memStore) Put(_ context.Context, key string, data []byte, contentType string) error {
	if m.putErr != nil {
		return m.putErr
	}
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	if m.types == nil {
		m.types = map[string]string{}
	}
	m.objects[key] = data
	m.types[key] = contentType
	m.puts = append(m.puts, key)
	return nil
}

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestGenerate(t *testing.T) {
	big := encodePNG(t, 1200, 600)
	small := encodePNG(t, 300, 200)

	t.Run("produces a webp rendition", func(t *testing.T) {
		store := &memStore{objects: map[string][]byte{"a.png": big}}
		res, err := Generate(context.Background(), store, "a.png", "image/png", 0)
		if err != nil {
			t.Fatal(err)
		}
		if res.ThumbKey != ObjectKey("a.png") || res.Unavailable {
			t.Fatalf("unexpected result: %+v", res)
		}
		if res.ContentType != ContentType {
			t.Fatalf("content type %q", res.ContentType)
		}
		if len(store.puts) != 1 || store.types[res.ThumbKey] != ContentType {
			t.Fatalf("puts: %v types: %v", store.puts, store.types)
		}
		img, err := webp.Decode(bytes.NewReader(store.objects[res.ThumbKey]))
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != DefaultMaxEdge {
			t.Fatalf("rendition width %d", img.Bounds().Dx())
		}
	})

	t.Run("unsupported mime is permanent", func(t *testing.T) {
		store := &memStore{objects: map[string][]byte{"a.webp": big}}
		res, err := Generate(context.Background(), store, "a.webp", "image/webp", 0)
		if err != nil || !res.Unavailable || len(store.puts) != 0 {
			t.Fatalf("res=%+v err=%v puts=%v", res, err, store.puts)
		}
	})

	t.Run("already small is permanent", func(t *testing.T) {
		store := &memStore{objects: map[string][]byte{"s.png": small}}
		res, err := Generate(context.Background(), store, "s.png", "image/png", DefaultMaxEdge)
		if err != nil || !res.Unavailable || len(store.puts) != 0 {
			t.Fatalf("res=%+v err=%v", res, err)
		}
	})

	t.Run("storage failures are retryable", func(t *testing.T) {
		// Get failure: caller should retry, so it must come back as err, not unavailable.
		if _, err := Generate(context.Background(), &memStore{getErr: errors.New("boom")}, "a.png", "image/png", 0); err == nil {
			t.Fatal("get failure must surface as error")
		}
		// Put failure: same.
		store := &memStore{objects: map[string][]byte{"a.png": big}, putErr: errors.New("boom")}
		if _, err := Generate(context.Background(), store, "a.png", "image/png", 0); err == nil {
			t.Fatal("put failure must surface as error")
		}
	})
}
