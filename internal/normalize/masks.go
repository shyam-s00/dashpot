package normalize

import "regexp"

var (
	// uuidToken runs before hexToken so a UUID isn't masked as generic hex.
	uuidToken = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)

	// isoDate covers ISO 8601 dates and datetimes.
	isoDate = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)?\b`)

	// unixTS is a heuristic (10-digit ints in 2017-2033's range); it can
	// false-positive on an ordinary large integer.
	unixTS = regexp.MustCompile(`\b1[5-9]\d{8}\b`)

	// tmpPath matches an ephemeral temp path up to the next whitespace/quote.
	tmpPath = regexp.MustCompile(`(?:/tmp|/var/folders)/\S*`)

	// hexToken catches 8+ hex digits, e.g. a commit hash or nonce.
	hexToken = regexp.MustCompile(`\b[0-9a-fA-F]{8,}\b`)
)

// Mask replaces volatile substrings (hashes, nonces, temp paths,
// timestamps) with stable placeholders.
func Mask(s string) string {
	s = uuidToken.ReplaceAllString(s, "<UUID>")
	s = isoDate.ReplaceAllString(s, "<TS>")
	s = unixTS.ReplaceAllString(s, "<TS>")
	s = tmpPath.ReplaceAllString(s, "<TMP_PATH>")
	s = hexToken.ReplaceAllString(s, "<HEX>")
	return s
}
