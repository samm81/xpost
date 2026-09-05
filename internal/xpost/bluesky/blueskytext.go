package bluesky

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf16"

	"github.com/bluesky-social/indigo/api/bsky"
)

const (
	blueskyShortURLPathLength = 15
	blueskyShortURLPrefix     = 13
)

// Bluesky's web composer shortens link facets before counting post graphemes.
var blueskyURLPattern = regexp.MustCompile(`(?i)https?://[^\s<>"']+`)

type preparedText struct {
	text   string
	facets []*bsky.RichtextFacet
}

func prepareText(text string) preparedText {
	matches := blueskyURLPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return preparedText{text: text}
	}

	var output strings.Builder
	output.Grow(len(text))
	facets := make([]*bsky.RichtextFacet, 0, len(matches))
	sourceEnd := 0

	for _, match := range matches {
		start, end := match[0], match[1]
		candidate := blueskyTrimURL(text[start:end])
		if candidate == "" || start < sourceEnd {
			continue
		}

		parsed, ok := blueskyParseURL(candidate)
		if !ok {
			continue
		}

		output.WriteString(text[sourceEnd:start])
		facetStart := output.Len()
		output.WriteString(blueskyShortURL(parsed))
		facetEnd := output.Len()
		facets = append(facets, blueskyLinkFacet(facetStart, facetEnd, candidate))
		sourceEnd = start + len(candidate)
	}

	output.WriteString(text[sourceEnd:])

	return preparedText{
		text:   output.String(),
		facets: facets,
	}
}

func blueskyTrimURL(candidate string) string {
	for candidate != "" {
		last := candidate[len(candidate)-1]
		switch last {
		case '.', ',', '!', ':', ';', '?':
			candidate = candidate[:len(candidate)-1]
		case ')':
			if strings.Count(candidate, "(") < strings.Count(candidate, ")") {
				candidate = candidate[:len(candidate)-1]
				continue
			}
			return candidate
		case ']':
			if strings.Count(candidate, "[") < strings.Count(candidate, "]") {
				candidate = candidate[:len(candidate)-1]
				continue
			}
			return candidate
		default:
			return candidate
		}
	}

	return candidate
}

func blueskyParseURL(candidate string) (*url.URL, bool) {
	parsed, err := url.Parse(candidate)
	if err != nil || parsed == nil || parsed.Host == "" {
		return nil, false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, false
	}

	return parsed, true
}

func blueskyShortURL(parsed *url.URL) string {
	path := parsed.EscapedPath()
	if path == "/" {
		path = ""
	}
	if parsed.ForceQuery || parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		path += "#" + parsed.EscapedFragment()
	}

	if blueskyUTF16Length(path) > blueskyShortURLPathLength {
		path = blueskyUTF16Prefix(path, blueskyShortURLPrefix) + "..."
	}

	return strings.ToLower(parsed.Host) + path
}

func blueskyUTF16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func blueskyUTF16Prefix(value string, length int) string {
	encoded := utf16.Encode([]rune(value))
	if len(encoded) <= length {
		return value
	}

	return string(utf16.Decode(encoded[:length]))
}

func blueskyLinkFacet(start, end int, uri string) *bsky.RichtextFacet {
	return &bsky.RichtextFacet{
		Index: &bsky.RichtextFacet_ByteSlice{
			ByteStart: int64(start),
			ByteEnd:   int64(end),
		},
		Features: []*bsky.RichtextFacet_Features_Elem{
			{
				RichtextFacet_Link: &bsky.RichtextFacet_Link{
					LexiconTypeID: "app.bsky.richtext.facet#link",
					Uri:           uri,
				},
			},
		},
	}
}
