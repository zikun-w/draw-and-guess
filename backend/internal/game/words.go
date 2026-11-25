package game

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

//go:embed words.txt
var embeddedWords string

var defaultWords = strings.Fields(embeddedWords)

// WordBank publishes complete, immutable word lists at turn boundaries.
type WordBank struct {
	current atomic.Pointer[[]string]
}

func (b *WordBank) snapshot() []string {
	if b != nil {
		if list := b.current.Load(); list != nil {
			return *list
		}
	}
	return defaultWords
}

func (b *WordBank) Replace(data []byte) (int, error) {
	var words []string
	seen := make(map[string]bool)
	for i, line := range strings.Split(string(data), "\n") {
		word := strings.TrimSpace(line)
		if word == "" {
			continue
		}
		length := utf8.RuneCountInString(word)
		if length < 2 || length > 8 || strings.IndexFunc(word, func(r rune) bool { return !unicode.Is(unicode.Han, r) }) >= 0 || seen[word] {
			return 0, fmt.Errorf("invalid or duplicate word at line %d", i+1)
		}
		words = append(words, word)
		seen[word] = true
	}
	if len(words) < 3 {
		return 0, errors.New("at least three distinct words are required")
	}
	b.current.Store(&words)
	return len(words), nil
}
