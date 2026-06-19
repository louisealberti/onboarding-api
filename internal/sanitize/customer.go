package sanitize

import "github.com/louisealberti/onboarding-api/internal/domain"

// ApplyToCustomer sanitizes every free-text field on c in place: first name,
// last name, and (when present) address street/city/state/postal code.
//
// Fields with their own strict format validation — Email, TaxID,
// CountryCode, Status — are intentionally left untouched here; HTML-escaping
// them could corrupt an otherwise-valid value (e.g. an email containing
// "&"). Phone fields (CountryCode, AreaCode, Number, Type) are digits/codes
// validated by the phone format itself and are likewise left untouched.
func ApplyToCustomer(c *domain.Customer) {
	if c == nil {
		return
	}

	c.FirstName = Text(c.FirstName, MaxTextFieldLength)
	c.LastName = Text(c.LastName, MaxTextFieldLength)

	if c.Address != nil {
		c.Address.Street = Text(c.Address.Street, MaxTextFieldLength)
		c.Address.City = Text(c.Address.City, MaxTextFieldLength)
		c.Address.State = Text(c.Address.State, MaxTextFieldLength)
		c.Address.PostalCode = Text(c.Address.PostalCode, 20)
	}
}
