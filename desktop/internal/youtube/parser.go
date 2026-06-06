// Package youtube parses links and resolves titles. It is a direct port of the Android app's
// YouTubeUrlParser / YouTubeMetadataClient so both backends behave identically.
package youtube

import "regexp"

var (
	bareID   = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	patterns = []*regexp.Regexp{
		regexp.MustCompile(`youtu\.be/([A-Za-z0-9_-]{11})`),
		regexp.MustCompile(`[?&]v=([A-Za-z0-9_-]{11})`),
		regexp.MustCompile(`/embed/([A-Za-z0-9_-]{11})`),
		regexp.MustCompile(`/shorts/([A-Za-z0-9_-]{11})`),
		regexp.MustCompile(`/live/([A-Za-z0-9_-]{11})`),
	}
)

// ExtractVideoID returns the 11-char video id from any supported link shape, or ("", false).
func ExtractVideoID(input string) (string, bool) {
	trimmed := trimSpace(input)
	if trimmed == "" {
		return "", false
	}
	if bareID.MatchString(trimmed) {
		return trimmed, true
	}
	for _, p := range patterns {
		if m := p.FindStringSubmatch(trimmed); m != nil {
			return m[1], true
		}
	}
	return "", false
}

// trimSpace avoids importing strings for one call and keeps the parser dependency-free.
func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }
