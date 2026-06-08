package email

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidEmail = errors.New("invalid email format")

	// RFC 5322 simplified — covers the vast majority of real-world email addresses
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// Validate checks whether the given email address has a valid format.
// It expects the email to already be trimmed and lowercased.
func Validate(email string) error {
	if strings.ContainsAny(email, " \t\n") {
		return ErrInvalidEmail
	}

	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}

	return nil
}
