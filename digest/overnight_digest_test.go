package digest

import (
	"strings"
	"testing"
)

func TestBuildOperatorDigestSeparatesReleaseBlockers(t *testing.T) {
	tests := []struct {
		name       string
		records    []Record
		wantAlert  bool
		wantActive string
	}{
		{
			name:      "critical diagnostic is held for attention",
			records:   []Record{{Kind: "model build", Summary: "compiler cache warmed"}, {Kind: "diagnostic", Summary: "release image signature missing", Severity: "critical"}},
			wantAlert: true, wantActive: "model build: compiler cache warmed",
		},
		{
			name:      "low diagnostic stays in activity",
			records:   []Record{{Kind: "release", Summary: "candidate published"}, {Kind: "diagnostic", Summary: "stale local cache", Severity: "low"}},
			wantAlert: false, wantActive: "diagnostic: stale local cache",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest := BuildOperatorDigest(tt.records)
			if strings.Contains(digest, "## Attention before release") != tt.wantAlert {
				t.Fatalf("attention section mismatch: %s", digest)
			}
			if !strings.Contains(digest, tt.wantActive) {
				t.Fatalf("activity missing %q: %s", tt.wantActive, digest)
			}
		})
	}
}
