package sanitize

import (
	"testing"

	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestApplyToCustomer_SanitizesNameFields(t *testing.T) {
	c := &domain.Customer{
		FirstName: "  <script>alert(1)</script>",
		LastName:  "Ferreira\x00",
	}

	ApplyToCustomer(c)

	assert.NotContains(t, c.FirstName, "<script>")
	assert.Equal(t, "Ferreira", c.LastName)
}

func TestApplyToCustomer_SanitizesAddressFields(t *testing.T) {
	c := &domain.Customer{
		FirstName: "Ana",
		LastName:  "Ferreira",
		Address: &domain.Address{
			Street:     `<img src=x onerror=alert(1)>`,
			City:       "  Curitiba  ",
			State:      "PR",
			PostalCode: "80000-000",
		},
	}

	ApplyToCustomer(c)

	assert.NotContains(t, c.Address.Street, "<img")
	assert.Equal(t, "Curitiba", c.Address.City)
	assert.Equal(t, "PR", c.Address.State)
	assert.Equal(t, "80000-000", c.Address.PostalCode)
}

func TestApplyToCustomer_NilAddressIsNoop(t *testing.T) {
	c := &domain.Customer{FirstName: "Ana", LastName: "Ferreira"}

	assert.NotPanics(t, func() { ApplyToCustomer(c) })
	assert.Nil(t, c.Address)
}

func TestApplyToCustomer_NilCustomerIsNoop(t *testing.T) {
	assert.NotPanics(t, func() { ApplyToCustomer(nil) })
}

func TestApplyToCustomer_DoesNotTouchEmailTaxIDCountryCodeStatus(t *testing.T) {
	c := &domain.Customer{
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Status:      "pending",
	}

	ApplyToCustomer(c)

	// These fields have their own format validation downstream;
	// sanitize must leave them exactly as received.
	assert.Equal(t, "ana@example.com", c.Email)
	assert.Equal(t, "52998224725", c.TaxID)
	assert.Equal(t, "BR", c.CountryCode)
	assert.Equal(t, "pending", c.Status)
}
