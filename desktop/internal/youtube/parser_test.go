package youtube

import "testing"

func TestExtractVideoID(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"standard watch", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", true},
		{"watch with params", "https://youtube.com/watch?list=abc&v=dQw4w9WgXcQ&t=30s", "dQw4w9WgXcQ", true},
		{"short link", "https://youtu.be/dQw4w9WgXcQ?si=xyz", "dQw4w9WgXcQ", true},
		{"shorts", "https://www.youtube.com/shorts/abcdefghijk", "abcdefghijk", true},
		{"embed", "https://www.youtube.com/embed/abcdefghijk", "abcdefghijk", true},
		{"live", "https://www.youtube.com/live/abcdefghijk", "abcdefghijk", true},
		{"bare id", "dQw4w9WgXcQ", "dQw4w9WgXcQ", true},
		{"non youtube", "https://example.com/watch?v=tooShort", "", false},
		{"empty", "", "", false},
		{"plain text", "just some text", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ExtractVideoID(tc.input)
			if ok != tc.ok || got != tc.want {
				t.Errorf("ExtractVideoID(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}
