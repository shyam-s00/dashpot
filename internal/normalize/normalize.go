// Package normalize turns a tool call's raw arguments into a
// deterministic byte key, so trivial variation (key order, spacing, a
// fresh nonce) collapses to the same bytes.
package normalize

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	// trailingNoise strips an agent's own retry annotation, e.g.
	// "# retry" or "# attempt 2", off the end of a string.
	trailingNoise = regexp.MustCompile(`(?i)\s*#\s*(retry|attempt\s*\d*)\s*$`)

	// consecutiveSpace collapses runs of spaces/tabs to one.
	consecutiveSpace = regexp.MustCompile(`[ \t]+`)
)

// Canonicalize turns raw `arguments` JSON into a deterministic byte
// form: keys sorted, whitespace and retry noise collapsed, volatile
// substrings masked (see masks.go).
//
// Malformed input still needs to produce some deterministic key, so it
// falls back to cleaning the raw bytes as a string instead of erroring.
func Canonicalize(raw []byte) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return []byte(cleanString(string(raw)))
	}
	out, err := json.Marshal(cleanValue(v))
	if err != nil {
		return []byte(cleanString(string(raw)))
	}
	return out
}

// cleanValue cleans every string leaf in a decoded JSON value.
// encoding/json already sorts map keys on Marshal.
func cleanValue(v any) any {
	switch t := v.(type) {
	case string:
		return cleanString(t)
	case map[string]any:
		for k, val := range t {
			t[k] = cleanValue(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = cleanValue(val)
		}
		return t
	default:
		return t
	}
}

// cleanString normalizes line endings, strips trailing noise, collapses
// whitespace, then masks volatile substrings.
func cleanString(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = trailingNoise.ReplaceAllString(s, "")
	s = strings.TrimRight(s, "; \t")
	s = consecutiveSpace.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return Mask(s)
}
