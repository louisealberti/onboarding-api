package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/louisealberti/onboarding-api/internal/validation/email"
	"github.com/louisealberti/onboarding-api/internal/validation/taxid"
)

// REGRAS DE NEGOCIO

type CustomerService struct {
	repo  domain.CustomerRepository // Dependency Injection (Interface)
	audit *AuditService
}

func NewCustomerService(repo domain.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

// WithAudit attaches an AuditService to record changes.
// Call this after NewCustomerService when audit logging is desired.
func (s *CustomerService) WithAudit(audit *AuditService) *CustomerService {
	s.audit = audit
	return s
}

func (s *CustomerService) CreateCustomer(ctx context.Context, c *domain.Customer) error {
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	if c.Email == "" {
		return ErrMissingEmail
	}
	if c.TaxID == "" {
		return ErrMissingTaxID
	}
	if c.CountryCode == "" {
		return ErrMissingCountryCode
	}
	if err := taxid.Validate(c.CountryCode, c.TaxID); err != nil { // ← aqui
		return ErrInvalidTaxID
	}
	if err := email.Validate(c.Email); err != nil {
		return ErrInvalidEmail
	}

	existing, err := s.repo.GetByEmail(ctx, c.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if existing != nil {
		return ErrDuplicatedEmail
	}

	now := time.Now().UTC()
	c.ID = uuid.New()
	c.CreatedAt = now
	c.UpdatedAt = now
	c.Status = "pending"
	c.Version = 1

	if c.Address != nil {
		c.Address.ID = uuid.New()
		c.Address.CustomerID = c.ID
		c.Address.CreatedAt = now
		c.Address.UpdatedAt = now
	}

	for i := range c.Phones {
		c.Phones[i].ID = uuid.New()
		c.Phones[i].CustomerID = c.ID
		c.Phones[i].CreatedAt = now
		c.Phones[i].UpdatedAt = now
	}

	if err := s.repo.CreateCustomer(ctx, c); err != nil {
		return err
	}
	if s.audit != nil {
		changedBy, _ := ctx.Value("changed_by").(string)
		if changedBy == "" {
			changedBy = "system"
		}
		s.audit.LogCreated(ctx, c, changedBy)
	}
	return nil
}

func (s *CustomerService) SearchCustomer(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCustomerNotRegistered
		}
		return nil, err
	}
	return customer, nil
}

func (s *CustomerService) SearchByTaxID(ctx context.Context, taxID string) (*domain.Customer, error) {
	customer, err := s.repo.GetByTaxID(ctx, taxID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCustomerNotRegistered
		}
		return nil, err
	}
	return customer, nil
}

func (s *CustomerService) UpdateCustomer(ctx context.Context, updatedCustomer *domain.Customer) error {
	updatedCustomer.Email = strings.ToLower(strings.TrimSpace(updatedCustomer.Email))
	if updatedCustomer.Email == "" {
		return ErrMissingEmail
	}
	if updatedCustomer.TaxID == "" {
		return ErrMissingTaxID
	}
	if updatedCustomer.CountryCode == "" {
		return ErrMissingCountryCode
	}
	if err := taxid.Validate(updatedCustomer.CountryCode, updatedCustomer.TaxID); err != nil {
		return ErrInvalidTaxID
	}
	if err := email.Validate(updatedCustomer.Email); err != nil {
		return ErrInvalidEmail
	}

	current, err := s.repo.GetByID(ctx, updatedCustomer.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCustomerNotRegistered
		}
		return err
	}

	now := time.Now().UTC()

	current.FirstName = updatedCustomer.FirstName
	current.LastName = updatedCustomer.LastName
	current.Email = updatedCustomer.Email
	current.TaxID = updatedCustomer.TaxID
	current.CountryCode = updatedCustomer.CountryCode
	current.Version = current.Version + 1
	current.UpdatedAt = now

	if updatedCustomer.Address != nil {
		if current.Address == nil {
			current.Address = updatedCustomer.Address
			current.Address.CustomerID = current.ID
			current.Address.CreatedAt = now
			current.Address.UpdatedAt = now
		} else {
			addressChanged := current.Address.Street != updatedCustomer.Address.Street ||
				current.Address.City != updatedCustomer.Address.City ||
				current.Address.State != updatedCustomer.Address.State ||
				current.Address.PostalCode != updatedCustomer.Address.PostalCode

			if addressChanged {
				current.Address.Street = updatedCustomer.Address.Street
				current.Address.City = updatedCustomer.Address.City
				current.Address.State = updatedCustomer.Address.State
				current.Address.PostalCode = updatedCustomer.Address.PostalCode
				current.Address.UpdatedAt = now
			}
		}
	}

	// Guarda os phones originais antes de sobrescrever
	currentPhones := current.Phones
	current.Phones = updatedCustomer.Phones

	for i := range current.Phones {
		if current.Phones[i].ID == uuid.Nil {
			existing := findPhoneByNumber(current.Phones[i].Number, currentPhones)
			if existing != nil {
				current.Phones[i].ID = existing.ID
				current.Phones[i].CreatedAt = existing.CreatedAt
				if phoneChanged(existing, &current.Phones[i]) {
					current.Phones[i].UpdatedAt = now
				} else {
					current.Phones[i].UpdatedAt = existing.UpdatedAt
				}
			} else {
				current.Phones[i].CreatedAt = now
				current.Phones[i].UpdatedAt = now
			}
		}
	}

	return s.repo.UpdateCustomer(ctx, current)
}
func (s *CustomerService) DeleteCustomer(ctx context.Context, id uuid.UUID) error {
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCustomerNotRegistered
		}
		return err
	}

	if customer.Status == "blocked" {
		return ErrCustomerIsBlocked
	}

	return s.repo.SoftDelete(ctx, id)
}

func (s *CustomerService) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus string) error {
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCustomerNotRegistered
		}
		return err
	}

	if !customer.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransition
	}

	oldStatus := customer.Status
	customer.Status = newStatus
	customer.Version = customer.Version + 1
	customer.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateCustomer(ctx, customer); err != nil {
		return err
	}
	if s.audit != nil {
		changedBy, _ := ctx.Value("changed_by").(string)
		if changedBy == "" {
			changedBy = "system"
		}
		s.audit.LogStatusChanged(ctx, customer.ID, oldStatus, newStatus, changedBy)
	}
	return nil
}

func findPhoneByNumber(number string, phones []domain.Phone) *domain.Phone {
	for i := range phones {
		if phones[i].Number == number {
			return &phones[i]
		}
	}
	return nil
}

func phoneChanged(existing, updated *domain.Phone) bool {
	return existing.CountryCode != updated.CountryCode ||
		existing.AreaCode != updated.AreaCode ||
		existing.Number != updated.Number ||
		existing.Type != updated.Type
}

func (s *CustomerService) ListCustomers(ctx context.Context, params domain.ListParams) (*domain.PaginatedCustomers, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}
	if params.Status != "" {
		validStatuses := map[string]bool{
			"pending": true, "approved": true, "active": true,
			"suspended": true, "blocked": true, "terminated": true,
		}
		if !validStatuses[params.Status] {
			return nil, ErrInvalidStatus
		}
	}
	return s.repo.ListCustomers(ctx, params)
}
