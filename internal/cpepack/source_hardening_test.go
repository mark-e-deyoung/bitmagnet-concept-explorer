package cpepack

import (
	"strings"
	"testing"
)

// Protect exact source-native byte identity and reject deceptive Git origin strings.
func TestInvalidUTF8NotSilentlyRecoded(t *testing.T) {
	c := Concept{ID: "public:bad", Definition: "bad" + string([]byte{0xff})}
	if _, err := ContentDigest([]Concept{c}); err == nil {
		t.Fatal("invalid UTF-8 silently normalized to replacement character")
	}
}
func TestUntrustedGitOriginRefused(t *testing.T) {
	for _, origin := range []string{"http://github.com/example/public", "https://github.com@example.net/repo", "https://github.com/example/public?token=secret", "file:///local/private/repo", "https://github.com:444/example/public"} {
		t.Run(origin, func(t *testing.T) {
			p, l := original()
			p[0].Repository = origin
			l.Packages[0].Repository = origin
			_, err := Build(p, l, []string{"test.public.literature"}, true)
			if err == nil || !strings.Contains(err.Error(), "invalid_git_origin") {
				t.Fatalf("untrusted origin admitted: %q / %v", origin, err)
			}
		})
	}
}
