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

func TestCreateCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
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
	})

	t.Run("missing e-mail", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		customer := newValidCustomer()
		customer.Email = ""

		err := svc.CreateCustomer(ctx, customer)

		assert.ErrorIs(t, err, ErrMissingEmail)
		repo.AssertNotCalled(t, "GetByEmail")
		repo.AssertNotCalled(t, "CreateCustomer")
	})

	t.Run("missing tax ID", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		customer := newValidCustomer()
		customer.TaxID = ""

		err := svc.CreateCustomer(ctx, customer)

		assert.ErrorIs(t, err, ErrMissingTaxID)
		repo.AssertNotCalled(t, "CreateCustomer")
	})

	t.Run("missing country code", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		customer := newValidCustomer()
		customer.CountryCode = ""

		err := svc.CreateCustomer(ctx, customer)

		assert.ErrorIs(t, err, ErrMissingCountryCode)
		repo.AssertNotCalled(t, "CreateCustomer")
	})

	t.Run("duplicate e-mail", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		customer := newValidCustomer()
		existing := newExistingCustomer()

		repo.On("GetByEmail", ctx, customer.Email).Return(existing, nil)

		err := svc.CreateCustomer(ctx, customer)

		assert.ErrorIs(t, err, ErrDuplicatedEmail)
		repo.AssertNotCalled(t, "CreateCustomer")
		repo.AssertExpectations(t)
	})

	t.Run("e-mail normalized", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		customer := newValidCustomer()
		customer.Email = "  ANA@EXAMPLE.COM  "

		repo.On("GetByEmail", ctx, "ana@example.com").Return(nil, sql.ErrNoRows)
		repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

		err := svc.CreateCustomer(ctx, customer)

		assert.NoError(t, err)
		assert.Equal(t, "ana@example.com", customer.Email)
		repo.AssertExpectations(t)
	})

	t.Run("with address and phones", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
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
	})

	t.Run("repo create error", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		customer := newValidCustomer()
		dbErr := errors.New("insert failed")

		repo.On("GetByEmail", ctx, customer.Email).Return(nil, sql.ErrNoRows)
		repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(dbErr)

		err := svc.CreateCustomer(ctx, customer)

		assert.ErrorIs(t, err, dbErr)
		repo.AssertExpectations(t)
	})

	t.Run("repository error on email check", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		customer := newValidCustomer()
		dbErr := errors.New("connection refused")

		repo.On("GetByEmail", ctx, customer.Email).Return(nil, dbErr)

		err := svc.CreateCustomer(ctx, customer)

		assert.ErrorIs(t, err, dbErr)
		repo.AssertNotCalled(t, "CreateCustomer")
		repo.AssertExpectations(t)
	})
}

func TestSearchCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)

		result, err := svc.SearchCustomer(ctx, existing.ID)

		assert.NoError(t, err)
		assert.Equal(t, existing.ID, result.ID)
		repo.AssertExpectations(t)
	})

	t.Run("customer not found", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()

		repo.On("GetByID", ctx, id).Return(nil, sql.ErrNoRows)

		result, err := svc.SearchCustomer(ctx, id)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrCustomerNotRegistered)
		repo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()
		dbErr := errors.New("connection timeout")

		repo.On("GetByID", ctx, id).Return(nil, dbErr)

		result, err := svc.SearchCustomer(ctx, id)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, dbErr)
		repo.AssertExpectations(t)
	})
}

func TestDeleteCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()
		existing.Status = "pending"

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("SoftDelete", ctx, existing.ID).Return(nil)

		err := svc.DeleteCustomer(ctx, existing.ID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("customer not found", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()

		repo.On("GetByID", ctx, id).Return(nil, sql.ErrNoRows)

		err := svc.DeleteCustomer(ctx, id)

		assert.ErrorIs(t, err, ErrCustomerNotRegistered)
		repo.AssertNotCalled(t, "SoftDelete")
		repo.AssertExpectations(t)
	})

	t.Run("customer blocked", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()
		existing.Status = "blocked"

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)

		err := svc.DeleteCustomer(ctx, existing.ID)

		assert.ErrorIs(t, err, ErrCustomerIsBlocked)
		repo.AssertNotCalled(t, "SoftDelete")
		repo.AssertExpectations(t)
	})

	t.Run("soft delete error", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()
		existing.Status = "pending"
		dbErr := errors.New("connection timeout")

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("SoftDelete", ctx, existing.ID).Return(dbErr)

		err := svc.DeleteCustomer(ctx, existing.ID)

		assert.ErrorIs(t, err, dbErr)
		repo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()
		dbErr := errors.New("connection timeout")

		repo.On("GetByID", ctx, id).Return(nil, dbErr)

		err := svc.DeleteCustomer(ctx, id)

		assert.ErrorIs(t, err, dbErr)
		repo.AssertNotCalled(t, "SoftDelete")
		repo.AssertExpectations(t)
	})
}

func TestUpdateCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

		var captured *domain.Customer
		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(*domain.Customer)
			}).
			Return(nil)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          existing.ID,
			FirstName:   "Ana Paula",
			LastName:    "Ferreira",
			Email:       "ana@example.com",
			TaxID:       "52998224725",
			CountryCode: "BR",
		})

		assert.NoError(t, err)
		assert.Equal(t, "Ana Paula", captured.FirstName)
		repo.AssertExpectations(t)
	})

	t.Run("customer not found", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()

		repo.On("GetByID", ctx, id).Return(nil, sql.ErrNoRows)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          id,
			Email:       "ana@example.com",
			TaxID:       "52998224725",
			CountryCode: "BR",
		})

		assert.ErrorIs(t, err, ErrCustomerNotRegistered)
		repo.AssertNotCalled(t, "UpdateCustomer")
		repo.AssertExpectations(t)
	})

	t.Run("version increment", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

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
		assert.Equal(t, 2, captured.Version)
		repo.AssertExpectations(t)
	})

	t.Run("address changed", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
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
	})

	t.Run("address not changed", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
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
	})

	t.Run("new address", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

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
	})

	t.Run("phone unchanged", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
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
				{CountryCode: "55", AreaCode: "41", Number: "991112233", Type: "mobile"},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, phoneID, captured.Phones[0].ID)
		assert.Equal(t, originalUpdatedAt, captured.Phones[0].UpdatedAt)
		repo.AssertExpectations(t)
	})

	t.Run("phone changed", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
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
				{CountryCode: "55", AreaCode: "41", Number: "999999999", Type: "mobile"},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, uuid.Nil, captured.Phones[0].ID)
		assert.True(t, captured.Phones[0].UpdatedAt.After(originalUpdatedAt))
		repo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()
		dbErr := errors.New("connection timeout")

		repo.On("GetByID", ctx, id).Return(nil, dbErr)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          id,
			Email:       "ana@example.com",
			TaxID:       "52998224725",
			CountryCode: "BR",
		})

		assert.ErrorIs(t, err, dbErr)
		repo.AssertNotCalled(t, "UpdateCustomer")
		repo.AssertExpectations(t)
	})
}

func TestCreateCustomer_TaxIDValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("tax ID validation", func(t *testing.T) {
		cases := []struct {
			name        string
			countryCode string
			taxID       string
			wantErr     bool
		}{
			{"valid CPF", "BR", "52998224725", false},
			{"invalid CPF", "BR", "00000000000", true},
			{"valid CNPJ", "BR", "11222333000181", false},
			{"invalid CNPJ", "BR", "11222333000182", true},
			{"valid SSN", "US", "123-45-6789", false},
			{"invalid SSN", "US", "000-45-6789", true},
			{"valid EIN", "US", "12-3456789", false},
			{"invalid EIN", "US", "00-3456789", true},
			{"valid NI", "GB", "AB123456C", false},
			{"invalid NI", "GB", "BG123456C", true},
			{"valid UTR", "GB", "1234567895", false},
			{"invalid UTR", "GB", "1234567890", true},
			{"unsupported country", "AR", "12345678901", true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				repo := new(MockCustomerRepository)
				svc := NewCustomerService(repo)
				customer := newValidCustomer()
				customer.CountryCode = tc.countryCode
				customer.TaxID = tc.taxID

				if !tc.wantErr {
					repo.On("GetByEmail", ctx, customer.Email).Return(nil, sql.ErrNoRows)
					repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)
				}

				err := svc.CreateCustomer(ctx, customer)

				if tc.wantErr {
					assert.ErrorIs(t, err, ErrInvalidTaxID)
					repo.AssertNotCalled(t, "GetByEmail")
					repo.AssertNotCalled(t, "CreateCustomer")
				} else {
					assert.NoError(t, err)
					repo.AssertExpectations(t)
				}
			})
		}
	})
}

func TestSearchByTaxID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

		repo.On("GetByTaxID", ctx, existing.TaxID).Return(existing, nil)

		result, err := svc.SearchByTaxID(ctx, existing.TaxID)

		assert.NoError(t, err)
		assert.Equal(t, existing.ID, result.ID)
		assert.Equal(t, existing.TaxID, result.TaxID)
		repo.AssertExpectations(t)
	})

	t.Run("customer not found", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)

		repo.On("GetByTaxID", ctx, "00000000000").Return(nil, sql.ErrNoRows)

		result, err := svc.SearchByTaxID(ctx, "00000000000")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrCustomerNotRegistered)
		repo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		dbErr := errors.New("connection timeout")

		repo.On("GetByTaxID", ctx, "00000000000").Return(nil, dbErr)

		result, err := svc.SearchByTaxID(ctx, "00000000000")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, dbErr)
		repo.AssertExpectations(t)
	})
}

func TestCreateCustomer_EmailValidation(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"valid email", "ana@example.com", nil},
		{"valid email with subdomain", "ana@mail.example.com.br", nil},
		{"valid email with plus", "ana+test@example.com", nil},
		{"missing @", "anaexample.com", ErrInvalidEmail},
		{"missing domain", "ana@", ErrInvalidEmail},
		{"missing TLD", "ana@example", ErrInvalidEmail},
		{"space inside email", "ana @example.com", ErrInvalidEmail}, // espaço interno sobrevive ao trim, regex rejeita
		{"only spaces", "   ", ErrMissingEmail},                     // trim → "", cai no check de empty
		// "ana@example..com" passes the simplified RFC5322 regex — not tested as invalid
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(MockCustomerRepository)
			svc := NewCustomerService(repo)
			customer := newValidCustomer()
			customer.Email = tc.email

			if tc.wantErr == nil {
				repo.On("GetByEmail", ctx, customer.Email).Return(nil, sql.ErrNoRows)
				repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)
			}

			err := svc.CreateCustomer(ctx, customer)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				repo.AssertNotCalled(t, "CreateCustomer")
			} else {
				assert.NoError(t, err)
				repo.AssertExpectations(t)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("success: pending → approved", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer() // status: "pending"

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("version is incremented", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

		var captured *domain.Customer
		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(*domain.Customer)
			}).
			Return(nil)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")

		assert.NoError(t, err)
		assert.Equal(t, 2, captured.Version)
		repo.AssertExpectations(t)
	})

	t.Run("status is updated", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

		var captured *domain.Customer
		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(*domain.Customer)
			}).
			Return(nil)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")

		assert.NoError(t, err)
		assert.Equal(t, "approved", captured.Status)
		repo.AssertExpectations(t)
	})

	t.Run("invalid transition retorna ErrInvalidStatusTransition", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer() // status: "pending"

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)

		// pending → active não é permitido
		err := svc.UpdateStatus(ctx, existing.ID, "active")

		assert.ErrorIs(t, err, ErrInvalidStatusTransition)
		repo.AssertNotCalled(t, "UpdateCustomer")
		repo.AssertExpectations(t)
	})

	t.Run("all valid transitions from pending", func(t *testing.T) {
		validTargets := []string{"approved", "blocked", "terminated"}
		for _, target := range validTargets {
			t.Run("pending → "+target, func(t *testing.T) {
				repo := new(MockCustomerRepository)
				svc := NewCustomerService(repo)
				existing := newExistingCustomer()

				repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
				repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

				err := svc.UpdateStatus(ctx, existing.ID, target)

				assert.NoError(t, err)
				repo.AssertExpectations(t)
			})
		}
	})

	t.Run("customer not found retorna ErrCustomerNotRegistered", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()

		repo.On("GetByID", ctx, id).Return(nil, sql.ErrNoRows)

		err := svc.UpdateStatus(ctx, id, "approved")

		assert.ErrorIs(t, err, ErrCustomerNotRegistered)
		repo.AssertNotCalled(t, "UpdateCustomer")
		repo.AssertExpectations(t)
	})

	t.Run("repository error no GetByID", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		id := uuid.New()
		dbErr := errors.New("connection timeout")

		repo.On("GetByID", ctx, id).Return(nil, dbErr)

		err := svc.UpdateStatus(ctx, id, "approved")

		assert.ErrorIs(t, err, dbErr)
		repo.AssertNotCalled(t, "UpdateCustomer")
		repo.AssertExpectations(t)
	})

	t.Run("repository error no UpdateCustomer", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()
		dbErr := errors.New("update failed")

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(dbErr)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")

		assert.ErrorIs(t, err, dbErr)
		repo.AssertExpectations(t)
	})
}

func TestUpdateCustomer_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("missing email retorna ErrMissingEmail", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          uuid.New(),
			TaxID:       "52998224725",
			CountryCode: "BR",
		})

		assert.ErrorIs(t, err, ErrMissingEmail)
		repo.AssertNotCalled(t, "GetByID")
	})

	t.Run("invalid email retorna ErrInvalidEmail", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          uuid.New(),
			Email:       "nao-e-email",
			TaxID:       "52998224725",
			CountryCode: "BR",
		})

		assert.ErrorIs(t, err, ErrInvalidEmail)
		repo.AssertNotCalled(t, "GetByID")
	})

	t.Run("missing taxId retorna ErrMissingTaxID", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          uuid.New(),
			Email:       "ana@example.com",
			CountryCode: "BR",
		})

		assert.ErrorIs(t, err, ErrMissingTaxID)
		repo.AssertNotCalled(t, "GetByID")
	})

	t.Run("invalid taxId retorna ErrInvalidTaxID", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          uuid.New(),
			Email:       "ana@example.com",
			TaxID:       "00000000000",
			CountryCode: "BR",
		})

		assert.ErrorIs(t, err, ErrInvalidTaxID)
		repo.AssertNotCalled(t, "GetByID")
	})

	t.Run("missing countryCode retorna ErrMissingCountryCode", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:    uuid.New(),
			Email: "ana@example.com",
			TaxID: "52998224725",
		})

		assert.ErrorIs(t, err, ErrMissingCountryCode)
		repo.AssertNotCalled(t, "GetByID")
	})

	t.Run("email normalizado antes da validação", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		existing := newExistingCustomer()

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

		err := svc.UpdateCustomer(ctx, &domain.Customer{
			ID:          existing.ID,
			Email:       "  ANA@EXAMPLE.COM  ",
			TaxID:       "52998224725",
			CountryCode: "BR",
			FirstName:   "Ana",
			LastName:    "Ferreira",
		})

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestListCustomers(t *testing.T) {
	ctx := context.Background()

	makeResult := func(total, page, limit int) *domain.PaginatedCustomers {
		customers := make([]domain.Customer, limit)
		return &domain.PaginatedCustomers{
			Data: customers,
			Meta: domain.PageMeta{Page: page, Limit: limit, Total: total, TotalPages: (total + limit - 1) / limit},
		}
	}

	t.Run("success with defaults", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		expected := makeResult(5, 1, 20)

		repo.On("ListCustomers", ctx, domain.ListParams{Page: 1, Limit: 20}).Return(expected, nil)

		result, err := svc.ListCustomers(ctx, domain.ListParams{Page: 1, Limit: 20})

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		repo.AssertExpectations(t)
	})

	t.Run("page < 1 defaults to 1", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		expected := makeResult(3, 1, 20)

		repo.On("ListCustomers", ctx, domain.ListParams{Page: 1, Limit: 20}).Return(expected, nil)

		result, err := svc.ListCustomers(ctx, domain.ListParams{Page: 0, Limit: 20})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		repo.AssertExpectations(t)
	})

	t.Run("limit > 100 defaults to 20", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		expected := makeResult(3, 1, 20)

		repo.On("ListCustomers", ctx, domain.ListParams{Page: 1, Limit: 20}).Return(expected, nil)

		result, err := svc.ListCustomers(ctx, domain.ListParams{Page: 1, Limit: 200})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		repo.AssertExpectations(t)
	})

	t.Run("valid status filter", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		expected := makeResult(2, 1, 20)

		repo.On("ListCustomers", ctx, domain.ListParams{Page: 1, Limit: 20, Status: "approved"}).Return(expected, nil)

		result, err := svc.ListCustomers(ctx, domain.ListParams{Page: 1, Limit: 20, Status: "approved"})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		repo.AssertExpectations(t)
	})

	t.Run("invalid status retorna ErrInvalidStatus", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)

		result, err := svc.ListCustomers(ctx, domain.ListParams{Page: 1, Limit: 20, Status: "nonexistent"})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidStatus)
		repo.AssertNotCalled(t, "ListCustomers")
	})

	t.Run("repository error", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo)
		dbErr := errors.New("db failure")

		repo.On("ListCustomers", ctx, domain.ListParams{Page: 1, Limit: 20}).Return(nil, dbErr)

		result, err := svc.ListCustomers(ctx, domain.ListParams{Page: 1, Limit: 20})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, dbErr)
		repo.AssertExpectations(t)
	})
}
