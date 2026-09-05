package twitter

import (
	"strings"
	"testing"

	"github.com/blacktop/xpost/internal/xpost"
)

func TestTwitterTextParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		length  int
		invalid bool
	}{
		{name: "plain text", text: "hello", length: 5},
		{name: "url is transformed to 23 characters", text: "Hi http://test.co", length: 26},
		{name: "long url is transformed to 23 characters", text: "https://github.com/samm81/a-long-path", length: 23},
		{name: "bare domain is transformed to 23 characters", text: "see example.com", length: 27},
		{name: "emoji sequences count as two characters", text: "H🐱☺👨‍👩‍👧‍👦", length: 7},
		{name: "emoji modifiers count as one sequence", text: "🙋🏽👨‍🎤", length: 4},
		{name: "nfc normalization", text: "ÁB", length: 2},
		{name: "cjk characters count as two", text: "故人", length: 4},
		{name: "invalid characters are reported", text: "bad\uFFFFtext", length: 9, invalid: true},
		{
			name:   "markdown-rendered github links use weighted urls",
			text:   "vibecoded a wrapper around xpost (https://github.com/samm81/xpost) (link to my fork, which also vibecoded changes)\nin order to be able to keep a small log about my ai experimentation on both twitter & bluesky: thought (https://github.com/samm81/thought)\n\nquite a rabbit-hole (yak-shave ?) to get here:",
			length: 283,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			length, invalid := twitterTextParse(test.text)
			if length != test.length {
				t.Fatalf("length = %d, want %d", length, test.length)
			}
			if invalid != test.invalid {
				t.Fatalf("invalid = %t, want %t", invalid, test.invalid)
			}
		})
	}
}

func TestTwitterTextParseURLPunctuation(t *testing.T) {
	t.Parallel()

	if length, _ := twitterTextParse("see https://example.com/path."); length != 28 {
		t.Fatalf("length = %d, want 28", length)
	}
	if length, _ := twitterTextParse("see https://example.com/(path)"); length != 27 {
		t.Fatalf("length = %d, want 27", length)
	}
}

func TestValidateRequestUsesWeightedURLLength(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("a", 258) + " https://example.com"
	err := validateRequest(xpost.Request{Message: text})
	if err == nil {
		t.Fatal("validateRequest() error = nil, want weighted URL length error")
	}
	if !strings.Contains(err.Error(), "282 weighted characters") {
		t.Fatalf("validateRequest() error = %q, want weighted length", err)
	}
}
