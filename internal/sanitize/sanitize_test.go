package sanitize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestText_TrimsWhitespace(t *testing.T) {
	got := Text("  Ana Ferreira  ", 0)
	assert.Equal(t, "Ana Ferreira", got)
}

func TestText_EscapesScriptTag(t *testing.T) {
	got := Text(`<script>alert('xss')</script>`, 0)
	assert.NotContains(t, got, "<script>")
	assert.Contains(t, got, "&lt;script&gt;")
}

func TestText_EscapesHTMLAttributeInjection(t *testing.T) {
	got := Text(`John" onmouseover="alert(1)`, 0)
	assert.NotContains(t, got, `"`)
	assert.Contains(t, got, "&#34;")
}

func TestText_StripsControlCharacters(t *testing.T) {
	got := Text("Ana\x00Ferreira\x07", 0)
	assert.Equal(t, "AnaFerreira", got)
}

func TestText_StripsNewlinesAndTabs(t *testing.T) {
	got := Text("Ana\nFerreira\t", 0)
	assert.Equal(t, "AnaFerreira", got)
}

func TestText_TruncatesToMaxLen(t *testing.T) {
	input := strings.Repeat("a", 300)
	got := Text(input, 50)
	assert.Len(t, got, 50)
}

func TestText_DefaultMaxLenWhenZero(t *testing.T) {
	input := strings.Repeat("a", 300)
	got := Text(input, 0)
	assert.Len(t, got, MaxTextFieldLength)
}

func TestText_TruncatesByRuneNotByte(t *testing.T) {
	// "á" is multi-byte in UTF-8; truncation must not split a rune in half.
	input := strings.Repeat("á", 10)
	got := Text(input, 5)
	assert.Equal(t, 5, len([]rune(got)))
}

func TestText_PreservesLegitimateContent(t *testing.T) {
	got := Text("José D'Ávila-Souza", 0)
	// Apostrophe gets HTML-escaped, which is expected and safe.
	assert.Contains(t, got, "José")
	assert.Contains(t, got, "Ávila-Souza")
}

func TestText_EmptyInput(t *testing.T) {
	got := Text("", 0)
	assert.Equal(t, "", got)
}

func TestText_OnlyWhitespaceInput(t *testing.T) {
	got := Text("   \t\n  ", 0)
	assert.Equal(t, "", got)
}
