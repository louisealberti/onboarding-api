package taxid

import "fmt"

// Validator defines the contract for tax ID validation per country.
type Validator interface {
	Validate(taxID string) error
}

// Validate dispatches tax ID validation based on the country code.
func Validate(countryCode, taxID string) error {
	validators := map[string]Validator{
		"BR": brValidator{},
		"US": usValidator{},
		"GB": gbValidator{},
	}

	v, ok := validators[countryCode]
	if !ok {
		return fmt.Errorf("no tax ID validator available for country %q", countryCode)
	}

	return v.Validate(taxID)
}
