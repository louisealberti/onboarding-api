package taxid

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidUTR     = errors.New("invalid UTR")
	ErrInvalidNI      = errors.New("invalid National Insurance number")
	ErrInvalidGBTaxID = errors.New("invalid GB tax ID: must be a UTR (10 digits) or NI number (e.g. AB123456C)")

	niRegex = regexp.MustCompile(`(?i)^[A-CEGHJ-PR-TW-Z]{2}\d{6}[A-D]$`)
)

type gbValidator struct{}

func (gbValidator) Validate(taxID string) error {
	digits := onlyDigits(taxID)
	normalized := strings.ToUpper(strings.ReplaceAll(taxID, " ", ""))

	switch {
	case len(digits) == 10 && taxID == digits:
		return validateUTR(digits)
	case niRegex.MatchString(normalized):
		return validateNI(normalized)
	default:
		return ErrInvalidGBTaxID
	}
}

func validateUTR(utr string) error {
	// UTR check digit algorithm (HMRC)
	weights := []int{2, 1, 3, 7, 6, 5, 4, 3, 2}
	sum := 0
	for i, w := range weights {
		sum += digitAt(utr, i) * w
	}
	remainder := sum % 11
	checkDigit := 11 - remainder
	if checkDigit == 10 {
		return ErrInvalidUTR
	}
	if checkDigit == 11 {
		checkDigit = 0
	}
	if checkDigit != digitAt(utr, 9) {
		return ErrInvalidUTR
	}
	return nil
}

func validateNI(ni string) error {
	// Invalid NI prefixes per HMRC
	invalidPrefixes := map[string]bool{
		"BG": true, "GB": true, "KN": true, "NK": true,
		"NT": true, "TN": true, "ZZ": true,
	}
	if invalidPrefixes[ni[:2]] {
		return ErrInvalidNI
	}
	return nil
}
