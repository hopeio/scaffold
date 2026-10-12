package sensitive

import (
	"errors"
	"testing"
)

func TestNoFilterPassthrough(t *testing.T) {
	Set(nil)
	if err := Check("any content"); err != nil {
		t.Fatalf("passthrough failed: %v", err)
	}
}

func TestWordListHit(t *testing.T) {
	Set(NewWordList([]string{"badword", " ad "}))
	defer Set(nil)

	if err := Check("this has BadWord inside"); !errors.Is(err, ErrSensitive) {
		t.Fatalf("want ErrSensitive, got %v", err)
	}
	// Matching is case-insensitive on the text as well as the word list.
	if err := Check("BUY NOW, AD inside"); !errors.Is(err, ErrSensitive) {
		t.Fatalf("want ErrSensitive for lower-cased word, got %v", err)
	}
	if err := Check("clean text"); err != nil {
		t.Fatalf("clean text blocked: %v", err)
	}
	if err := Check(""); err != nil {
		t.Fatalf("empty text blocked: %v", err)
	}
}

func TestNewWordListDropsBlank(t *testing.T) {
	wl := NewWordList([]string{"", "   ", "Keep"}).(*wordList)
	if len(wl.words) != 1 || wl.words[0] != "keep" {
		t.Fatalf("want single lower-cased word, got %#v", wl.words)
	}
}
