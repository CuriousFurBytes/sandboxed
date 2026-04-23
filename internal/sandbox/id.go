package sandbox

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
)

var nonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// ID derives a deterministic container name from an absolute directory path.
// Algorithm mirrors the original shell implementation:
//
//	sbx-<sanitized-basename[:40]>-<sha256(absPath)[:10]>
func ID(absPath string) string {
	sum := sha256.Sum256([]byte(absPath))
	hash := fmt.Sprintf("%x", sum)[:10]

	base := filepath.Base(absPath)
	base = nonAlphanumRe.ReplaceAllString(base, "-")
	if len(base) > 40 {
		base = base[:40]
	}

	return fmt.Sprintf("sbx-%s-%s", base, hash)
}
