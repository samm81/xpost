package bluesky

import (
	"strings"
	"testing"

	"github.com/blacktop/xpost/internal/xpost"
	"github.com/rivo/uniseg"
)

func TestPrepareTextShortensLinkFacets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		text      string
		wantText  string
		wantURI   string
		wantStart int64
		wantEnd   int64
	}{
		{
			name:      "short path",
			text:      "see https://github.com/samm81/xpost.",
			wantText:  "see github.com/samm81/xpost.",
			wantURI:   "https://github.com/samm81/xpost",
			wantStart: 4,
			wantEnd:   27,
		},
		{
			name:      "unicode prefix uses byte offsets",
			text:      "🐱 https://example.com",
			wantText:  "🐱 example.com",
			wantURI:   "https://example.com",
			wantStart: 5,
			wantEnd:   16,
		},
		{
			name:      "long path is truncated",
			text:      "read https://example.com/12345678901234567890.",
			wantText:  "read example.com/123456789012....",
			wantURI:   "https://example.com/12345678901234567890",
			wantStart: 5,
			wantEnd:   32,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepared := prepareText(test.text)
			if prepared.text != test.wantText {
				t.Fatalf("text = %q, want %q", prepared.text, test.wantText)
			}
			if len(prepared.facets) != 1 {
				t.Fatalf("facet count = %d, want 1", len(prepared.facets))
			}

			facet := prepared.facets[0]
			if facet.Index.ByteStart != test.wantStart || facet.Index.ByteEnd != test.wantEnd {
				t.Fatalf("facet range = [%d, %d), want [%d, %d)", facet.Index.ByteStart, facet.Index.ByteEnd, test.wantStart, test.wantEnd)
			}
			link := facet.Features[0].RichtextFacet_Link
			if link.Uri != test.wantURI {
				t.Fatalf("facet URI = %q, want %q", link.Uri, test.wantURI)
			}
		})
	}
}

func TestPrepareTextPreservesMarkdownPunctuation(t *testing.T) {
	t.Parallel()

	prepared := prepareText("[xpost](https://github.com/samm81/xpost)")
	if prepared.text != "[xpost](github.com/samm81/xpost)" {
		t.Fatalf("text = %q", prepared.text)
	}
}

func TestPrepareTextMatchesBlueskyComposerSample(t *testing.T) {
	t.Parallel()

	text := "vibecoded a wrapper around [xpost](https://github.com/samm81/xpost) (link to my fork, which also vibecoded changes) in order to be able to keep a small log about my ai experimentation on both twitter & bluesky: [thought](https://github.com/samm81/thought)\n\nquite a rabbit-hole (yak-shave ?) to get here:\n"
	prepared := prepareText(text)
	if count := uniseg.GraphemeClusterCount(prepared.text); count != 288 {
		t.Fatalf("shortened grapheme count = %d, want 288", count)
	}
	if len(prepared.facets) != 2 {
		t.Fatalf("facet count = %d, want 2", len(prepared.facets))
	}
}

func TestValidateRequestUsesShortenedURLLength(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("a", 285) + " https://example.com"
	if err := validateRequest(xpost.Request{Message: text}); err != nil {
		t.Fatalf("validateRequest() error = %v, want nil", err)
	}

	prepared := prepareText(text)
	if count := uniseg.GraphemeClusterCount(prepared.text); count != 297 {
		t.Fatalf("shortened grapheme count = %d, want 297", count)
	}
}

func TestValidateRequestRejectsShortenedTextOverLimit(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("a", 291) + " https://example.com"
	err := validateRequest(xpost.Request{Message: text})
	if err == nil {
		t.Fatal("validateRequest() error = nil, want length error")
	}
	if !strings.Contains(err.Error(), "303 graphemes") {
		t.Fatalf("validateRequest() error = %q, want shortened grapheme count", err)
	}
}
