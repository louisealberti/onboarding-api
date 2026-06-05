// Contém a struct Cliente e a interface ClienteRepository
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID         uuid.UUID  `json:"id"`
	CustomerID uuid.UUID  `json:"customerId"`
	Street     string     `json:"street"`
	City       string     `json:"city"`
	State      string     `json:"state"`
	PostalCode string     `json:"postalCode"`
	CreatedAt  time.Time  `json:"createdAt"`           // For Audit
	UpdatedAt  time.Time  `json:"updatedAt"`           // For Audit
	DeletedAt  *time.Time `json:"deletedAt,omitempty"` // Soft Delete
}

type Phone struct {
	ID          uuid.UUID  `json:"id"`
	CustomerID  uuid.UUID  `json:"customerId"`
	CountryCode string     `json:"countryCode"`
	AreaCode    string     `json:"areaCode"`
	Number      string     `json:"number"`
	Type        string     `json:"type"`                // eg: "mobile", "landline"
	CreatedAt   time.Time  `json:"createdAt"`           // For Audit
	UpdatedAt   time.Time  `json:"updatedAt"`           // For Audit
	DeletedAt   *time.Time `json:"deletedAt,omitempty"` // Soft Delete
}

type Customer struct {
	ID          uuid.UUID  `json:"id"`
	FirstName   string     `json:"firstName"`
	LastName    string     `json:"lastName"`
	Email       string     `json:"email"`
	TaxID       string     `json:"taxId"`       // Unique national identifier for tax purposes
	CountryCode string     `json:"countryCode"` // Tax residency / Tax jurisdiction (e.g., "BR", "US")
	Status      string     `json:"status"`      // eg: "pending", "approved", "active", "blocked", "terminated"
	Version     int        `json:"version"`     // Concurrency Control
	Address     *Address   `json:"address,omitempty"`
	Phones      []Phone    `json:"phones,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`           // For Audit
	UpdatedAt   time.Time  `json:"updatedAt"`           // For Audit
	DeletedAt   *time.Time `json:"deletedAt,omitempty"` // Soft Delete
}

// CustomerRepository
type CustomerRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	GetByEmail(ctx context.Context, email string) (*Customer, error)
	CreateCustomer(ctx context.Context, customer *Customer) error
	UpdateCustomer(ctx context.Context, customer *Customer) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
