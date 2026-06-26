package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/louisealberti/onboarding-api/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateCustomer_IncrementsCustomersCreatedTotal(t *testing.T) {
	ctx := context.Background()
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	customer := newValidCustomer()

	repo.On("GetByEmail", ctx, customer.Email).Return(nil, sql.ErrNoRows)
	repo.On("CreateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

	before := testutil.ToFloat64(metrics.CustomersCreatedTotal)

	err := svc.CreateCustomer(ctx, customer)

	assert.NoError(t, err)
	after := testutil.ToFloat64(metrics.CustomersCreatedTotal)
	assert.Equal(t, before+1, after, "expected CustomersCreatedTotal to increment by exactly 1")
}

func TestCreateCustomer_DoesNotIncrementMetricOnFailure(t *testing.T) {
	ctx := context.Background()
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	customer := newValidCustomer()

	existing := newExistingCustomer()
	repo.On("GetByEmail", ctx, customer.Email).Return(existing, nil) // duplicate email -> CreateCustomer fails

	before := testutil.ToFloat64(metrics.CustomersCreatedTotal)

	err := svc.CreateCustomer(ctx, customer)

	assert.Error(t, err)
	after := testutil.ToFloat64(metrics.CustomersCreatedTotal)
	assert.Equal(t, before, after, "expected CustomersCreatedTotal to stay unchanged when creation fails")
	repo.AssertNotCalled(t, "CreateCustomer")
}

func TestUpdateStatus_IncrementsStatusTransitionMetricWithCorrectLabels(t *testing.T) {
	ctx := context.Background()
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	existing := newExistingCustomer() // status: "pending"

	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
	repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

	before := testutil.ToFloat64(metrics.CustomerStatusTransitionsTotal.WithLabelValues("pending", "approved"))

	err := svc.UpdateStatus(ctx, existing.ID, "approved")

	assert.NoError(t, err)
	after := testutil.ToFloat64(metrics.CustomerStatusTransitionsTotal.WithLabelValues("pending", "approved"))
	assert.Equal(t, before+1, after, "expected the pending->approved counter to increment by exactly 1")
}

func TestUpdateStatus_DoesNotIncrementMetricOnInvalidTransition(t *testing.T) {
	ctx := context.Background()
	repo := new(MockCustomerRepository)
	svc := NewCustomerService(repo)
	existing := newExistingCustomer() // status: "pending"

	repo.On("GetByID", ctx, existing.ID).Return(existing, nil)

	before := testutil.ToFloat64(metrics.CustomerStatusTransitionsTotal.WithLabelValues("pending", "active"))

	// pending -> active is not a valid transition
	err := svc.UpdateStatus(ctx, existing.ID, "active")

	assert.Error(t, err)
	after := testutil.ToFloat64(metrics.CustomerStatusTransitionsTotal.WithLabelValues("pending", "active"))
	assert.Equal(t, before, after, "expected the metric to stay unchanged for a rejected transition")
	repo.AssertNotCalled(t, "UpdateCustomer")
}
