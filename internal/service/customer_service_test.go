package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// helpers to avoid repetition
func newValidCustomer() *domain.Customer {
	return &domain.Customer{
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
	}
}

func newExistingCustomer() *domain.Customer {
	return &domain.Customer{
		ID:          uuid.New(),
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Status:      "pending",
		Version:     1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

// =====================
// CreateCustomer
// =====================

func TestCreateCustomer_Success(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()

	repo.On("GetByEmail", ctx, customer.Email).Return(nil, sql.ErrNoRows)
	repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

	err := svc.CreateCustomer(ctx, customer)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, customer.ID)
	assert.Equal(t, "pending", customer.Status)
	assert.Equal(t, 1, customer.Version)
	assert.Equal(t, "ana@example.com", customer.Email)
	repo.AssertExpectations(t)
}

func TestCreateCustomer_MissingEmail(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	customer.Email = ""

	err := svc.CreateCustomer(ctx, customer)

	assert.ErrorIs(t, err, ErrMissingEmail)
	repo.AssertNotCalled(t, "GetByEmail")
	repo.AssertNotCalled(t, "CreateCustomer")
}

func TestCreateCustomer_MissingTaxID(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	customer.TaxID = ""

	err := svc.CreateCustomer(ctx, customer)

	assert.ErrorIs(t, err, ErrMissingTaxID)
	repo.AssertNotCalled(t, "CreateCustomer")
}

func TestCreateCustomer_DuplicateEmail(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	existing := newExistingCustomer()

	repo.On("GetByEmail", ctx, customer.Email).Return(existing, nil)

	err := svc.CreateCustomer(ctx, customer)

	assert.ErrorIs(t, err, ErrDuplicatedEmail)
	repo.AssertNotCalled(t, "CreateCustomer")
	repo.AssertExpectations(t)
}

func TestCreateCustomer_EmailNormalized(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	customer.Email = "  ANA@EXAMPLE.COM  "

	repo.On("GetByEmail", ctx, "ana@example.com").Return(nil, sql.ErrNoRows)
	repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

	err := svc.CreateCustomer(ctx, customer)

	assert.NoError(t, err)
	assert.Equal(t, "ana@example.com", customer.Email)
	repo.AssertExpectations(t)
}

func TestCreateCustomer_MissingCountryCode(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	customer.CountryCode = ""

	err := svc.CreateCustomer(ctx, customer)

	assert.ErrorIs(t, err, ErrMissingCountryCode)
	repo.AssertNotCalled(t, "CreateCustomer")
}

func TestCreateCustomer_RepositoryError(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	dbErr := errors.New("connection refused")

	repo.On("GetByEmail", ctx, customer.Email).Return(nil, dbErr)

	err := svc.CreateCustomer(ctx, customer)

	assert.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "CreateCustomer")
	repo.AssertExpectations(t)
}

// =====================
// SearchCustomer
// =====================

func TestSearchCustomer_Success(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()

	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)

	result, err := svc.SearchCustomer(ctx, existing.ID)

	assert.NoError(t, err)
	assert.Equal(t, existing.ID, result.ID)
	repo.AssertExpectations(t)
}

func TestSearchCustomer_NotFound(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, sql.ErrNoRows)

	result, err := svc.SearchCustomer(ctx, id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrCustomerNotRegistered)
	repo.AssertExpectations(t)
}

// =====================
// DeleteCustomer
// =====================

func TestDeleteCustomer_Success(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()
	existing.Status = "pending"

	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("SoftDelete", ctx, existing.ID).Return(nil)

	err := svc.DeleteCustomer(ctx, existing.ID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteCustomer_NotFound(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, sql.ErrNoRows)

	err := svc.DeleteCustomer(ctx, id)

	assert.ErrorIs(t, err, ErrCustomerNotRegistered)
	repo.AssertNotCalled(t, "SoftDelete")
	repo.AssertExpectations(t)
}

func TestDeleteCustomer_Blocked(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()
	existing.Status = "blocked"

	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)

	err := svc.DeleteCustomer(ctx, existing.ID)

	assert.ErrorIs(t, err, ErrCustomerIsBlocked)
	repo.AssertNotCalled(t, "SoftDelete")
	repo.AssertExpectations(t)
}

// =====================
// UpdateCustomer
// =====================

func TestUpdateCustomer_Success(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()

	updated := &domain.Customer{
		ID:          existing.ID,
		FirstName:   "Ana Paula",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
	}

	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

	err := svc.UpdateCustomer(ctx, updated)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_NotFound(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, sql.ErrNoRows)

	err := svc.UpdateCustomer(ctx, &domain.Customer{ID: id})

	assert.ErrorIs(t, err, ErrCustomerNotRegistered)
	repo.AssertNotCalled(t, "UpdateCustomer")
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_VersionIncrement(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer() // Version: 1

	var captured *domain.Customer
	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domain.Customer)
		}).
		Return(nil)

	err := svc.UpdateCustomer(ctx, &domain.Customer{
		ID:          existing.ID,
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, captured.Version) // version incrementou
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_AddressChanged(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()
	existing.Address = &domain.Address{
		ID:         uuid.New(),
		CustomerID: existing.ID,
		Street:     "Rua A, 100",
		City:       "Curitiba",
		State:      "PR",
		PostalCode: "80000-000",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	originalAddressUpdatedAt := existing.Address.UpdatedAt

	var captured *domain.Customer
	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domain.Customer)
		}).
		Return(nil)

	err := svc.UpdateCustomer(ctx, &domain.Customer{
		ID:          existing.ID,
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Address: &domain.Address{
			Street:     "Rua B, 200",
			City:       "Curitiba",
			State:      "PR",
			PostalCode: "80000-000",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "Rua B, 200", captured.Address.Street)
	assert.True(t, captured.Address.UpdatedAt.After(originalAddressUpdatedAt))
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_AddressNotChanged(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()
	originalUpdatedAt := time.Now().UTC().Add(-1 * time.Hour)
	existing.Address = &domain.Address{
		ID:         uuid.New(),
		CustomerID: existing.ID,
		Street:     "Rua A, 100",
		City:       "Curitiba",
		State:      "PR",
		PostalCode: "80000-000",
		CreatedAt:  originalUpdatedAt,
		UpdatedAt:  originalUpdatedAt,
	}

	var captured *domain.Customer
	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domain.Customer)
		}).
		Return(nil)

	err := svc.UpdateCustomer(ctx, &domain.Customer{
		ID:          existing.ID,
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Address: &domain.Address{
			Street:     "Rua A, 100",
			City:       "Curitiba",
			State:      "PR",
			PostalCode: "80000-000",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, originalUpdatedAt, captured.Address.UpdatedAt)
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_NewAddress(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer() // sem address

	var captured *domain.Customer
	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domain.Customer)
		}).
		Return(nil)

	err := svc.UpdateCustomer(ctx, &domain.Customer{
		ID:          existing.ID,
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Address: &domain.Address{
			Street:     "Rua Nova, 1",
			City:       "Curitiba",
			State:      "PR",
			PostalCode: "80000-000",
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, captured.Address)
	assert.Equal(t, existing.ID, captured.Address.CustomerID)
	assert.NotZero(t, captured.Address.CreatedAt)
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_PhoneUnchanged(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()
	phoneID := uuid.New()
	originalUpdatedAt := time.Now().UTC().Add(-1 * time.Hour)
	existing.Phones = []domain.Phone{
		{
			ID:          phoneID,
			CustomerID:  existing.ID,
			CountryCode: "55",
			AreaCode:    "41",
			Number:      "991112233",
			Type:        "mobile",
			CreatedAt:   originalUpdatedAt,
			UpdatedAt:   originalUpdatedAt,
		},
	}

	var captured *domain.Customer
	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domain.Customer)
		}).
		Return(nil)

	err := svc.UpdateCustomer(ctx, &domain.Customer{
		ID:          existing.ID,
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Phones: []domain.Phone{
			{
				CountryCode: "55",
				AreaCode:    "41",
				Number:      "991112233",
				Type:        "mobile",
			},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, phoneID, captured.Phones[0].ID)
	assert.Equal(t, originalUpdatedAt, captured.Phones[0].UpdatedAt)
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_PhoneChanged(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	existing := newExistingCustomer()
	phoneID := uuid.New()
	originalUpdatedAt := time.Now().UTC().Add(-1 * time.Hour)
	existing.Phones = []domain.Phone{
		{
			ID:          phoneID,
			CustomerID:  existing.ID,
			CountryCode: "55",
			AreaCode:    "41",
			Number:      "991112233",
			Type:        "mobile",
			CreatedAt:   originalUpdatedAt,
			UpdatedAt:   originalUpdatedAt,
		},
	}

	var captured *domain.Customer
	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domain.Customer)
		}).
		Return(nil)

	err := svc.UpdateCustomer(ctx, &domain.Customer{
		ID:          existing.ID,
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Phones: []domain.Phone{
			{ // número diferente — phone novo
				CountryCode: "55",
				AreaCode:    "41",
				Number:      "999999999",
				Type:        "mobile",
			},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, uuid.Nil, captured.Phones[0].ID)
	assert.True(t, captured.Phones[0].UpdatedAt.After(originalUpdatedAt))
	repo.AssertExpectations(t)
}

// =====================
// Missing coverage cases
// =====================

func TestCreateCustomer_WithAddressAndPhones(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	customer.Address = &domain.Address{
		Street:     "Rua das Flores, 42",
		City:       "Curitiba",
		State:      "PR",
		PostalCode: "80000-000",
	}
	customer.Phones = []domain.Phone{
		{CountryCode: "55", AreaCode: "41", Number: "991112233", Type: "mobile"},
	}

	repo.On("GetByEmail", ctx, customer.Email).Return(nil, sql.ErrNoRows)
	repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

	err := svc.CreateCustomer(ctx, customer)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, customer.Address.ID)
	assert.Equal(t, customer.ID, customer.Address.CustomerID)
	assert.NotZero(t, customer.Address.CreatedAt)
	assert.NotEqual(t, uuid.Nil, customer.Phones[0].ID)
	assert.Equal(t, customer.ID, customer.Phones[0].CustomerID)
	repo.AssertExpectations(t)
}

func TestCreateCustomer_RepoCreateError(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := newValidCustomer()
	dbErr := errors.New("insert failed")

	repo.On("GetByEmail", ctx, customer.Email).Return(nil, sql.ErrNoRows)
	repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(dbErr)

	err := svc.CreateCustomer(ctx, customer)

	assert.ErrorIs(t, err, dbErr)
	repo.AssertExpectations(t)
}

func TestSearchCustomer_RepositoryError(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()
	dbErr := errors.New("connection timeout")

	repo.On("GetByID", ctx, id).Return(nil, dbErr)

	result, err := svc.SearchCustomer(ctx, id)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, dbErr)
	repo.AssertExpectations(t)
}

func TestDeleteCustomer_RepositoryError(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()
	dbErr := errors.New("connection timeout")

	repo.On("GetByID", ctx, id).Return(nil, dbErr)

	err := svc.DeleteCustomer(ctx, id)

	assert.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "SoftDelete")
	repo.AssertExpectations(t)
}

func TestUpdateCustomer_RepositoryError(t *testing.T) {
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()
	dbErr := errors.New("connection timeout")

	repo.On("GetByID", ctx, id).Return(nil, dbErr)

	err := svc.UpdateCustomer(ctx, &domain.Customer{ID: id})

	assert.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "UpdateCustomer")
	repo.AssertExpectations(t)
}
