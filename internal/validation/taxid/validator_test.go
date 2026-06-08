package taxid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate_UnsupportedCountry(t *testing.T) {
	err := Validate("AR", "12345678901")
	assert.ErrorContains(t, err, "no tax ID validator available")
}

// ── Brazil ─────────────────────────────────────────────────────────────────

func TestBR_CPF(t *testing.T) {
	t.Run("valid CPF", func(t *testing.T) {
		assert.NoError(t, Validate("BR", "52998224725"))
	})

	t.Run("valid CPF with formatting", func(t *testing.T) {
		assert.NoError(t, Validate("BR", "529.982.247-25"))
	})

	t.Run("invalid CPF - wrong check digit", func(t *testing.T) {
		assert.ErrorIs(t, Validate("BR", "52998224724"), ErrInvalidCPF)
	})

	t.Run("invalid CPF - all same digits", func(t *testing.T) {
		assert.ErrorIs(t, Validate("BR", "11111111111"), ErrInvalidCPF)
	})

	t.Run("invalid CPF - wrong length", func(t *testing.T) {
		assert.ErrorIs(t, Validate("BR", "1234567890"), ErrInvalidBRTaxID)
	})
}

func TestBR_CNPJ(t *testing.T) {
	t.Run("valid CNPJ", func(t *testing.T) {
		assert.NoError(t, Validate("BR", "11222333000181"))
	})

	t.Run("valid CNPJ with formatting", func(t *testing.T) {
		assert.NoError(t, Validate("BR", "11.222.333/0001-81"))
	})

	t.Run("invalid CNPJ - wrong check digit", func(t *testing.T) {
		assert.ErrorIs(t, Validate("BR", "11222333000182"), ErrInvalidCNPJ)
	})

	t.Run("invalid CNPJ - all same digits", func(t *testing.T) {
		assert.ErrorIs(t, Validate("BR", "00000000000000"), ErrInvalidCNPJ)
	})
}

// ── United States ───────────────────────────────────────────────────────────

func TestUS_SSN(t *testing.T) {
	t.Run("valid SSN", func(t *testing.T) {
		assert.NoError(t, Validate("US", "123-45-6789"))
	})

	t.Run("invalid SSN - area 000", func(t *testing.T) {
		assert.ErrorIs(t, Validate("US", "000-45-6789"), ErrInvalidSSN)
	})

	t.Run("invalid SSN - area 666", func(t *testing.T) {
		assert.ErrorIs(t, Validate("US", "666-45-6789"), ErrInvalidSSN)
	})

	t.Run("invalid SSN - area starting with 9", func(t *testing.T) {
		assert.ErrorIs(t, Validate("US", "900-45-6789"), ErrInvalidSSN)
	})

	t.Run("invalid SSN - group 00", func(t *testing.T) {
		assert.ErrorIs(t, Validate("US", "123-00-6789"), ErrInvalidSSN)
	})

	t.Run("invalid SSN - serial 0000", func(t *testing.T) {
		assert.ErrorIs(t, Validate("US", "123-45-0000"), ErrInvalidSSN)
	})
}

func TestUS_EIN(t *testing.T) {
	t.Run("valid EIN", func(t *testing.T) {
		assert.NoError(t, Validate("US", "12-3456789"))
	})

	t.Run("invalid EIN - bad prefix", func(t *testing.T) {
		assert.ErrorIs(t, Validate("US", "00-3456789"), ErrInvalidEIN)
	})

	t.Run("invalid format", func(t *testing.T) {
		assert.ErrorIs(t, Validate("US", "123456789"), ErrInvalidUSTaxID)
	})
}

// ── United Kingdom ──────────────────────────────────────────────────────────

func TestGB_NI(t *testing.T) {
	t.Run("valid NI number", func(t *testing.T) {
		assert.NoError(t, Validate("GB", "AB123456C"))
	})

	t.Run("valid NI number with spaces", func(t *testing.T) {
		assert.NoError(t, Validate("GB", "AB 12 34 56 C"))
	})

	t.Run("invalid NI - bad prefix BG", func(t *testing.T) {
		assert.ErrorIs(t, Validate("GB", "BG123456C"), ErrInvalidNI)
	})

	t.Run("invalid format", func(t *testing.T) {
		assert.ErrorIs(t, Validate("GB", "ABCDEFGHI"), ErrInvalidGBTaxID)
	})
}

func TestGB_UTR(t *testing.T) {
	t.Run("valid UTR", func(t *testing.T) {
		assert.NoError(t, Validate("GB", "1234567895"))
	})

	t.Run("invalid UTR - wrong check digit", func(t *testing.T) {
		assert.ErrorIs(t, Validate("GB", "1234567890"), ErrInvalidUTR)
	})
}
