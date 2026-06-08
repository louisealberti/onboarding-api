package taxid

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidSSN     = errors.New("invalid SSN")
	ErrInvalidEIN     = errors.New("invalid EIN")
	ErrInvalidUSTaxID = errors.New("invalid US tax ID: must be a SSN (XXX-XX-XXXX) or EIN (XX-XXXXXXX)")

	ssnRegex = regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)
	einRegex = regexp.MustCompile(`^\d{2}-\d{7}$`)
)

type usValidator struct{}

func (usValidator) Validate(taxID string) error {
	switch {
	case ssnRegex.MatchString(taxID):
		return validateSSN(taxID)
	case einRegex.MatchString(taxID):
		return validateEIN(taxID)
	default:
		return ErrInvalidUSTaxID
	}
}

func validateSSN(ssn string) error {
	digits := onlyDigits(ssn)
	area := digits[:3]
	group := digits[3:5]
	serial := digits[5:]

	if area == "000" || area == "666" || area[0] == '9' {
		return ErrInvalidSSN
	}
	if group == "00" {
		return ErrInvalidSSN
	}
	if serial == "0000" {
		return ErrInvalidSSN
	}
	return nil
}

func validateEIN(ein string) error {
	digits := onlyDigits(ein)
	// Invalid EIN prefixes per IRS
	invalidPrefixes := map[string]bool{
		"00": true, "07": true, "08": true, "09": true,
		"17": true, "18": true, "19": true, "28": true,
		"29": true, "49": true, "69": true, "70": true,
		"78": true, "79": true, "89": true, "96": true, "97": true,
	}
	if invalidPrefixes[digits[:2]] {
		return ErrInvalidEIN
	}
	return nil
}
