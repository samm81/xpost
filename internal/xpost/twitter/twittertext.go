package twitter

import (
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/norm"
)

// These values mirror twitter-text's config/v3.json.
const (
	twitterTextMaxLength      = 280
	twitterTextDefaultWeight  = 2
	twitterTextURLWeight      = 23
	twitterTextMaxURLLength   = 4096
	twitterTextMaxDomainLabel = 63
)

var twitterTextEmojiBaseRanges = [...]struct {
	start rune
	end   rune
}{
	{start: 0x1F000, end: 0x1FAFF},
	{start: 0x2190, end: 0x21FF},
	{start: 0x2300, end: 0x27BF},
	{start: 0x2B00, end: 0x2BFF},
	{start: 0x3030, end: 0x303D},
}

var twitterTextURLCandidatePattern = regexp.MustCompile(
	`(?i)(https?://|www\.)[^\s<>"']+|[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+(:[0-9]+)?(/[^\s<>"']+)?`,
)

type twitterTextURLRange struct {
	start int
	end   int
}

func twitterTextParse(text string) (length int, invalid bool) {
	text = norm.NFC.String(text)
	invalid = strings.ContainsAny(text, "\uFFFE\uFEFF\uFFFF")
	urls := twitterTextURLRanges(text)

	graphemes := uniseg.NewGraphemes(text)
	urlIndex := 0
	for graphemes.Next() {
		start, end := graphemes.Positions()
		for urlIndex < len(urls) && urls[urlIndex].end <= start {
			urlIndex++
		}

		if urlIndex < len(urls) && start >= urls[urlIndex].start && end <= urls[urlIndex].end {
			if start == urls[urlIndex].start {
				length += twitterTextURLWeight
			}

			continue
		}

		cluster := graphemes.Str()
		length += twitterTextClusterWeight(cluster)
	}

	return length, invalid
}

func twitterTextClusterWeight(cluster string) int {
	if twitterTextIsEmojiCluster(cluster) {
		return twitterTextDefaultWeight
	}

	weight := 0
	for _, character := range cluster {
		weight += twitterTextCharacterWeight(character)
	}

	return weight
}

func twitterTextCharacterWeight(character rune) int {
	switch {
	case character >= 0 && character <= 4351:
		return 1
	case character >= 8192 && character <= 8205:
		return 1
	case character >= 8208 && character <= 8223:
		return 1
	case character >= 8242 && character <= 8247:
		return 1
	default:
		return twitterTextDefaultWeight
	}
}

func twitterTextIsEmojiCluster(cluster string) bool {
	runes := []rune(cluster)
	if len(runes) < 2 {
		return false
	}

	hasEmojiBase := false
	hasRegionalIndicator := true
	for _, character := range runes {
		if twitterTextIsEmojiBase(character) {
			hasEmojiBase = true
		}
		if character < 0x1F1E6 || character > 0x1F1FF {
			hasRegionalIndicator = false
		}
	}
	if !hasEmojiBase {
		return false
	}

	if strings.ContainsRune(cluster, '\uFE0F') || strings.ContainsRune(cluster, '\u20E3') {
		return true
	}
	if strings.ContainsRune(cluster, '\u200D') || twitterTextHasEmojiModifier(runes) {
		return true
	}
	if hasRegionalIndicator {
		return true
	}

	return uniseg.StringWidth(cluster) == 2
}

func twitterTextIsEmojiBase(character rune) bool {
	for _, bounds := range twitterTextEmojiBaseRanges {
		if character >= bounds.start && character <= bounds.end {
			return true
		}
	}

	return character == 0x00A9 || character == 0x00AE || character == 0x2122 ||
		character == '#' || character == '*' || character >= '0' && character <= '9'
}

func twitterTextHasEmojiModifier(runes []rune) bool {
	for _, character := range runes {
		if character >= 0x1F3FB && character <= 0x1F3FF {
			return true
		}
	}

	return false
}

func twitterTextURLRanges(text string) []twitterTextURLRange {
	matches := twitterTextURLCandidatePattern.FindAllStringIndex(text, -1)
	ranges := make([]twitterTextURLRange, 0, len(matches))

	for _, match := range matches {
		start, end := match[0], match[1]
		candidate := twitterTextTrimURLCandidate(text[start:end])
		if candidate == "" {
			continue
		}

		candidateEnd := start + len(candidate)
		bare := !twitterTextHasScheme(candidate)
		if !twitterTextValidURLPreceding(text, start, bare) || !twitterTextValidURL(candidate) {
			continue
		}
		if len(ranges) > 0 && start < ranges[len(ranges)-1].end {
			continue
		}

		ranges = append(ranges, twitterTextURLRange{start: start, end: candidateEnd})
	}

	return ranges
}

func twitterTextTrimURLCandidate(candidate string) string {
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

func twitterTextValidURLPreceding(text string, start int, bare bool) bool {
	if start == 0 {
		return true
	}

	character, _ := utf8.DecodeLastRuneInString(text[:start])
	if twitterTextASCIIAlphaNumeric(character) || strings.ContainsRune("@＠$#＃", character) {
		return false
	}
	if bare && strings.ContainsRune("-_./", character) {
		return false
	}

	return true
}

func twitterTextValidURL(candidate string) bool {
	if len(candidate) > twitterTextMaxURLLength {
		return false
	}

	parsed, ok := twitterTextParseURL(candidate)
	if !ok {
		return false
	}

	return twitterTextValidHost(parsed.Hostname())
}

func twitterTextParseURL(candidate string) (*url.URL, bool) {
	value := candidate
	if !twitterTextHasScheme(value) {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, false
	}

	return parsed, true
}

func twitterTextValidHost(host string) bool {
	if host == "" {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	if !strings.Contains(host, ".") {
		return false
	}

	return twitterTextValidDomain(host)
}

func twitterTextValidDomain(host string) bool {
	labels := strings.Split(host, ".")
	allNumeric := true
	for _, label := range labels {
		if !twitterTextValidDomainLabel(label) {
			return false
		}
		if !twitterTextDomainLabelNumeric(label) {
			allNumeric = false
		}
	}

	return !allNumeric
}

func twitterTextValidDomainLabel(label string) bool {
	return label != "" && len(label) <= twitterTextMaxDomainLabel &&
		!strings.HasPrefix(label, "-") && !strings.HasSuffix(label, "-")
}

func twitterTextDomainLabelNumeric(label string) bool {
	for _, character := range label {
		if !unicode.IsDigit(character) {
			return false
		}
	}

	return true
}

func twitterTextHasScheme(value string) bool {
	value = strings.ToLower(value)
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func twitterTextASCIIAlphaNumeric(character rune) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}
