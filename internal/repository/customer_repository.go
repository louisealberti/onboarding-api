package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/google/uuid"
)

type CustomerRepository struct {
	DB *sql.DB // DB connection
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{DB: db}
}

func (r *CustomerRepository) CreateCustomer(ctx context.Context, customer *domain.Customer) error {

	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	customerQuery := `
        INSERT INTO customers (id, first_name, last_name, email, tax_id, country_code, status, version, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
	_, err = tx.ExecContext(ctx, customerQuery,
		customer.ID, customer.FirstName, customer.LastName, customer.Email,
		customer.TaxID, customer.CountryCode, customer.Status, customer.Version,
		customer.CreatedAt, customer.UpdatedAt,
	)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if customer.Address != nil {
		addressQuery := `
            INSERT INTO addresses (id, customer_id, street, city, state, postal_code, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        `
		_, err = tx.ExecContext(ctx, addressQuery,
			customer.Address.ID, customer.ID, customer.Address.Street,
			customer.Address.City, customer.Address.State, customer.Address.PostalCode, customer.Address.CreatedAt, customer.Address.UpdatedAt)
		if err != nil {
			return err
		}
	}

	phoneQuery := `
        INSERT INTO phones (id, customer_id, country_code, area_code, number, type, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
	for _, phone := range customer.Phones {
		_, err = tx.ExecContext(ctx, phoneQuery,
			phone.ID, customer.ID, phone.CountryCode,
			phone.AreaCode, phone.Number, phone.Type, phone.CreatedAt, phone.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *CustomerRepository) UpdateCustomer(ctx context.Context, customer *domain.Customer) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	customerQuery := `
    	UPDATE customers 
    	SET first_name = $1, last_name = $2, email = $3, tax_id = $4, country_code = $5, version = $6, updated_at = $7
    	WHERE id = $8 AND deleted_at IS NULL
	`
	_, err = tx.ExecContext(ctx, customerQuery,
		customer.FirstName, customer.LastName, customer.Email,
		customer.TaxID, customer.CountryCode, customer.Version, customer.UpdatedAt, customer.ID,
	)
	if err != nil {
		return err
	}

	if customer.Address != nil {
		addressQuery := `
    		UPDATE addresses 
    		SET street = $1, city = $2, state = $3, postal_code = $4, updated_at = $5
    		WHERE customer_id = $6
		`
		_, err = tx.ExecContext(ctx, addressQuery,
			customer.Address.Street, customer.Address.City, customer.Address.State,
			customer.Address.PostalCode, customer.Address.UpdatedAt, customer.ID,
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx, `UPDATE phones SET deleted_at = NOW() WHERE customer_id = $1`, customer.ID)
	if err != nil {
		return err
	}

	for _, phone := range customer.Phones {
		if phone.ID == uuid.Nil {
			phone.ID = uuid.New()
			insertPhoneQuery := `
    			INSERT INTO phones (id, customer_id, country_code, area_code, number, type, created_at, updated_at)
    			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`
			_, err = tx.ExecContext(ctx, insertPhoneQuery,
				phone.ID, customer.ID, phone.CountryCode,
				phone.AreaCode, phone.Number, phone.Type,
				phone.CreatedAt, phone.UpdatedAt,
			)

			if err != nil {
				return err
			}
		} else {
			updatePhoneQuery := `
        		UPDATE phones
        		SET country_code = $1, area_code = $2, number = $3, type = $4, updated_at = $5, deleted_at = NULL
        		WHERE id = $6 AND customer_id = $7
    		`
			_, err = tx.ExecContext(ctx, updatePhoneQuery,
				phone.CountryCode, phone.AreaCode, phone.Number, phone.Type, phone.UpdatedAt,
				phone.ID, customer.ID,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *CustomerRepository) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	customer := &domain.Customer{}

	query := `
        SELECT id, first_name, last_name, email, tax_id, country_code,
               status, version, created_at, updated_at
        FROM customers
        WHERE email = $1 AND deleted_at IS NULL
    `
	err := r.DB.QueryRowContext(ctx, query, email).Scan(
		&customer.ID,
		&customer.FirstName,
		&customer.LastName,
		&customer.Email,
		&customer.TaxID,
		&customer.CountryCode,
		&customer.Status,
		&customer.Version,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return customer, nil
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	customer := &domain.Customer{}

	customerQuery := `
        SELECT id, first_name, last_name, email, tax_id, country_code,
               status, version, created_at, updated_at
        FROM customers
        WHERE id = $1 AND deleted_at IS NULL
    `
	err := r.DB.QueryRowContext(ctx, customerQuery, id).Scan(
		&customer.ID,
		&customer.FirstName,
		&customer.LastName,
		&customer.Email,
		&customer.TaxID,
		&customer.CountryCode,
		&customer.Status,
		&customer.Version,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	address := &domain.Address{}
	addressQuery := `
        SELECT id, customer_id, street, city, state, postal_code, created_at, updated_at
        FROM addresses
        WHERE customer_id = $1 AND deleted_at IS NULL
    `
	err = r.DB.QueryRowContext(ctx, addressQuery, id).Scan(
		&address.ID,
		&address.CustomerID,
		&address.Street,
		&address.City,
		&address.State,
		&address.PostalCode,
		&address.CreatedAt,
		&address.UpdatedAt,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		customer.Address = address
	}

	phoneQuery := `
        SELECT id, customer_id, country_code, area_code, number, type, created_at, updated_at
        FROM phones
        WHERE customer_id = $1 AND deleted_at IS NULL
    `
	rows, err := r.DB.QueryContext(ctx, phoneQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var phone domain.Phone
		if err := rows.Scan(
			&phone.ID,
			&phone.CustomerID,
			&phone.CountryCode,
			&phone.AreaCode,
			&phone.Number,
			&phone.Type,
			&phone.CreatedAt,
			&phone.UpdatedAt,
		); err != nil {
			return nil, err
		}
		customer.Phones = append(customer.Phones, phone)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return customer, nil
}

func (r *CustomerRepository) GetByTaxID(ctx context.Context, taxID string) (*domain.Customer, error) {
    customer := &domain.Customer{}

    query := `
        SELECT id, first_name, last_name, email, tax_id, country_code,
               status, version, created_at, updated_at
        FROM customers
        WHERE tax_id = $1 AND deleted_at IS NULL
    `
    err := r.DB.QueryRowContext(ctx, query, taxID).Scan(
        &customer.ID,
        &customer.FirstName,
        &customer.LastName,
        &customer.Email,
        &customer.TaxID,
        &customer.CountryCode,
        &customer.Status,
        &customer.Version,
        &customer.CreatedAt,
        &customer.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }

    return customer, nil
}

func (r *CustomerRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
        UPDATE customers 
        SET deleted_at = NOW(), status = 'terminated', updated_at = NOW() 
        WHERE id = $1 AND deleted_at IS NULL
    `

	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
