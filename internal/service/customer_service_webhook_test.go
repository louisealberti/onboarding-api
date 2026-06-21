package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/louisealberti/onboarding-api/internal/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestUpdateStatus_Webhook covers the integration between UpdateStatus and
// an attached webhook.Notifier: the webhook must be triggered on a
// successful transition, and — critically — its outcome must never affect
// the result UpdateStatus returns to its caller (the fire-and-forget
// contract described on webhook.Notifier.NotifyStatusChanged).
func TestUpdateStatus_Webhook(t *testing.T) {
	ctx := context.Background()

	t.Run("triggers webhook on successful transition", func(t *testing.T) {
		var received int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&received, 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		repo := new(MockCustomerRepository)
		notifier := webhook.NewNotifier(srv.URL, "", nil)
		svc := NewCustomerService(repo).WithWebhook(notifier)
		existing := newExistingCustomer() // status: "pending"

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")
		assert.NoError(t, err)

		// The webhook fires in a background goroutine; give it a moment
		// to land rather than asserting immediately after UpdateStatus
		// returns (which is the whole point of it being asynchronous).
		assert.Eventually(t, func() bool {
			return atomic.LoadInt32(&received) == 1
		}, time.Second, 10*time.Millisecond, "expected exactly one webhook delivery")
	})

	t.Run("UpdateStatus succeeds even if the webhook destination is unreachable", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		// Port 0: nothing listening, connection refused immediately —
		// simulates a webhook destination that is completely down.
		notifier := webhook.NewNotifier("http://127.0.0.1:0", "", nil).WithBackoff(time.Millisecond)
		svc := NewCustomerService(repo).WithWebhook(notifier)
		existing := newExistingCustomer()

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")

		// The status transition already succeeded in the repository by
		// the time the webhook is even attempted — an unreachable webhook
		// must not turn that into an error.
		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("UpdateStatus succeeds even if the webhook destination returns 500 repeatedly", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		repo := new(MockCustomerRepository)
		notifier := webhook.NewNotifier(srv.URL, "", nil).WithBackoff(time.Millisecond)
		svc := NewCustomerService(repo).WithWebhook(notifier)
		existing := newExistingCustomer()

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("no webhook attached: UpdateStatus behaves exactly as before", func(t *testing.T) {
		repo := new(MockCustomerRepository)
		svc := NewCustomerService(repo) // no .WithWebhook(...)
		existing := newExistingCustomer()

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)
		repo.On("UpdateCustomer", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil)

		err := svc.UpdateStatus(ctx, existing.ID, "approved")

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("invalid transition: webhook is never triggered", func(t *testing.T) {
		var calls int32
		var mu sync.Mutex
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			calls++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		repo := new(MockCustomerRepository)
		notifier := webhook.NewNotifier(srv.URL, "", nil)
		svc := NewCustomerService(repo).WithWebhook(notifier)
		existing := newExistingCustomer() // status: "pending"

		repo.On("GetByID", ctx, existing.ID).Return(existing, nil)

		// pending → active is not a valid transition
		err := svc.UpdateStatus(ctx, existing.ID, "active")
		assert.Error(t, err)

		// Give any (incorrectly fired) async call a chance to land before
		// asserting it never happened.
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		assert.Equal(t, int32(0), calls)
		mu.Unlock()

		repo.AssertNotCalled(t, "UpdateCustomer")
	})
}
