package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	t.Run("valid email", func(t *testing.T) {
		assert.NoError(t, Validate("ana@example.com"))
	})

	t.Run("valid email with subdomain", func(t *testing.T) {
		assert.NoError(t, Validate("ana@mail.example.com"))
	})

	t.Run("valid email with plus tag", func(t *testing.T) {
		assert.NoError(t, Validate("ana+test@example.com"))
	})

	t.Run("valid email with dots", func(t *testing.T) {
		assert.NoError(t, Validate("ana.ferreira@example.com"))
	})

	t.Run("missing @", func(t *testing.T) {
		assert.ErrorIs(t, Validate("anaexample.com"), ErrInvalidEmail)
	})

	t.Run("missing domain", func(t *testing.T) {
		assert.ErrorIs(t, Validate("ana@"), ErrInvalidEmail)
	})

	t.Run("missing TLD", func(t *testing.T) {
		assert.ErrorIs(t, Validate("ana@example"), ErrInvalidEmail)
	})

	t.Run("empty string", func(t *testing.T) {
		assert.ErrorIs(t, Validate(""), ErrInvalidEmail)
	})

	t.Run("contains space", func(t *testing.T) {
		assert.ErrorIs(t, Validate("ana @example.com"), ErrInvalidEmail)
	})

	t.Run("double @", func(t *testing.T) {
		assert.ErrorIs(t, Validate("ana@@example.com"), ErrInvalidEmail)
	})
}
