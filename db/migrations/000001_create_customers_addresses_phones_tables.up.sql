-- 1. Customer Table
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    
    -- TaxID: The unique national identifier used for tax purposes (e.g., CPF in Brazil, SSN in the US, NIF in Portugal).
    tax_id VARCHAR(50) UNIQUE NOT NULL,
    
    -- CountryCode: Identifies the customer's tax residency / tax jurisdiction (ISO 3166-1 alpha-2).
    country_code VARCHAR(2) NOT NULL,
    
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 2. Address Table - One-to-one relationship with Customer
CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL UNIQUE, -- UNIQUE garantees it's 1 to 1
    street VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_customer_address FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
);

-- 3. Phone Table - One-to-many relationship with Customer
CREATE TABLE IF NOT EXISTS phones (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL, -- Without UNIQUE, allows several phones
    country_code VARCHAR(5) NOT NULL,
    area_code VARCHAR(5) NOT NULL,
    number VARCHAR(20) NOT NULL,
    type VARCHAR(30) NOT NULL DEFAULT 'mobile',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_customer_phone FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
);

-- High Performance Index (Fraud Prevention and Fast Searches)
CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_tax_id ON customers(tax_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_addresses_customer_id ON addresses(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_phones_customer_id ON phones(customer_id) WHERE deleted_at IS NULL;