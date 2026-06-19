// Package sanitize provides defense-in-depth helpers for cleaning free-text
// user input before it is persisted or returned by the API.
//
// SQL injection is already mitigated at the repository layer, since every
// query uses parameterized placeholders ($1, $2, ...) via database/sql.
// This package focuses on the risks that parameterized queries do NOT cover:
//
//   - XSS: free-text fields (names, addresses) are stored as-is and returned
//     verbatim in JSON responses. Any future consumer that renders this data
//     as HTML (the planned React backoffice, for instance) would be exposed
//     to stored XSS unless the payload is neutralized first.
//   - Control characters: invisible characters (NUL, escape sequences, etc.)
//     can break logs, terminal output, and downstream parsers.
//   - Oversized payloads: unbounded strings are a cheap denial-of-service
//     vector and a sign of malformed or malicious input.
package sanitize

import (
	"html"
	"regexp"
	"strings"
)

// MaxTextFieldLength is the default cap applied to free-text fields such as
// first name, last name, street, city, and state. Chosen generously above
// any legitimate value while still blocking abusive payloads.
const MaxTextFieldLength = 255

// controlChars matches ASCII control characters (0x00–0x1F and 0x7F),
// excluding nothing — newlines and tabs inside a name/address are not
// legitimate either, so they are stripped along with the rest.
var controlChars = regexp.MustCompile(`[\x00-\x1F\x7F]`)

// Text sanitizes a single free-text field:
//  1. trims leading/trailing whitespace
//  2. strips control characters
//  3. HTML-escapes the result, neutralizing <script>, onerror=, etc.
//  4. truncates to maxLen runes (0 means MaxTextFieldLength)
//
// The HTML-escaped form is what gets persisted. This is a deliberate choice:
// it means stored data is always safe to render in an HTML context, at the
// cost of escaped entities (e.g. &amp;) showing up if the field is consumed
// as plain text elsewhere. For this API's fields (names, addresses) that
// trade-off is acceptable and matches common defense-in-depth practice.
func Text(input string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = MaxTextFieldLength
	}

	s := strings.TrimSpace(input)
	s = controlChars.ReplaceAllString(s, "")
	s = html.EscapeString(s)

	if runes := []rune(s); len(runes) > maxLen {
		s = string(runes[:maxLen])
	}

	return s
}
