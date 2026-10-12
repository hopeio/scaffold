// Package sensitive is a pluggable content filter hook: passthrough by default,
// a word list or an external moderation service when wired up.
//
// Pure logic with no global dependency, so it stays testable. Assembly happens
// in main (enabled when a word list is configured, zero cost otherwise).
// Call sites are UGC entry points shared across domains: moments, comments,
// chat messages.
package sensitive

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"sync/atomic"
)

// ErrSensitive means the text hit a blocked word; the service layer turns it
// into an InvalidArgument carrying an i18n key.
var ErrSensitive = errors.New("content contains blocked words")

// Filter checks a piece of content; a non-nil return means rejected.
type Filter interface {
	Check(text string) error
}

var current atomic.Pointer[Filter]

// Set registers the active filter (wired at startup; nil restores passthrough).
func Set(f Filter) {
	if f == nil {
		current.Store(nil)
		return
	}
	current.Store(&f)
}

// Check runs the active filter; passthrough when none is registered.
func Check(text string) error {
	p := current.Load()
	if p == nil {
		return nil
	}
	return (*p).Check(text)
}

// wordList does naive substring matching: good enough for word lists in the
// hundreds-to-thousands and UGC up to a few thousand runes. Swap in an
// Aho-Corasick implementation for larger dictionaries, keeping Filter stable.
type wordList struct {
	words []string
}

func (w *wordList) Check(text string) error {
	if text == "" {
		return nil
	}
	lower := strings.ToLower(text)
	for _, word := range w.words {
		if strings.Contains(lower, word) {
			return ErrSensitive
		}
	}
	return nil
}

// NewWordList builds a filter from words; each is trimmed and lower-cased,
// empty entries dropped.
func NewWordList(words []string) Filter {
	out := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			out = append(out, w)
		}
	}
	return &wordList{words: out}
}

// LoadWordFile reads a word list file: one word per line, '#' starts a comment.
func LoadWordFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var words []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		words = append(words, line)
	}
	return words, sc.Err()
}
