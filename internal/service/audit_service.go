package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/louisealberti/onboarding-api/internal/repository"
)

// AuditService records audit log entries for customer operations.
type AuditService struct {
	repo *repository.AuditRepository
}

func NewAuditService(repo *repository.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// LogCreated records a customer creation event.
func (s *AuditService) LogCreated(ctx context.Context, customer *domain.Customer, changedBy string) {
	newValue, err := json.Marshal(customer)
	if err != nil {
		return
	}
	_ = s.repo.Save(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		CustomerID: customer.ID,
		Action:     domain.AuditActionCreated,
		NewValue:   newValue,
		ChangedBy:  changedBy,
		CreatedAt:  time.Now().UTC(),
	})
}

// LogUpdated records a customer data update event.
func (s *AuditService) LogUpdated(ctx context.Context, old, new *domain.Customer, changedBy string) {
	oldValue, err := json.Marshal(old)
	if err != nil {
		return
	}
	newValue, err := json.Marshal(new)
	if err != nil {
		return
	}
	_ = s.repo.Save(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		CustomerID: new.ID,
		Action:     domain.AuditActionUpdated,
		OldValue:   oldValue,
		NewValue:   newValue,
		ChangedBy:  changedBy,
		CreatedAt:  time.Now().UTC(),
	})
}

// LogStatusChanged records a status transition event.
func (s *AuditService) LogStatusChanged(ctx context.Context, customerID uuid.UUID, oldStatus, newStatus, changedBy string) {
	oldValue, _ := json.Marshal(map[string]string{"status": oldStatus})
	newValue, _ := json.Marshal(map[string]string{"status": newStatus})
	_ = s.repo.Save(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		CustomerID: customerID,
		Action:     domain.AuditActionStatusChanged,
		OldValue:   oldValue,
		NewValue:   newValue,
		ChangedBy:  changedBy,
		CreatedAt:  time.Now().UTC(),
	})
}

// ListByCustomer returns all audit entries for a customer.
func (s *AuditService) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.AuditLog, error) {
	return s.repo.ListByCustomer(ctx, customerID)
}
