package service

import "errors"

var (
	ErrMissingEmail            = errors.New("customer email is missing")
	ErrMissingTaxID            = errors.New("customer tax id is missing")
	ErrMissingCountryCode      = errors.New("customer country code is missing")
	ErrDuplicatedEmail         = errors.New("customer's email is already in use")
	ErrInvalidEmail            = errors.New("customer's email is invalid")
	ErrDuplicatedTaxID         = errors.New("customer's tax id is already in use")
	ErrCustomerNotRegistered   = errors.New("customer does not exist")
	ErrCustomerIsBlocked       = errors.New("customer's status is 'blocked',no action is allowed")
	ErrInvalidTaxID            = errors.New("invalid tax ID for the given country")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidStatus           = errors.New("invalid status value")
)
