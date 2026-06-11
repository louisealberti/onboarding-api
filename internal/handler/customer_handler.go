package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/louisealberti/onboarding-api/internal/service"
)

// Gin Routes

type CustomerHandler struct {
	srv *service.CustomerService
}

func NewCustomerHandler(srv *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{srv: srv}
}

// CreateCustomer godoc
//
//	@Summary		Create a customer
//	@Description	Creates a new customer. TaxId must be valid for the given countryCode (BR: CPF/CNPJ, US: SSN/EIN, GB: NI/UTR).
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			customer	body		domain.Customer	true	"Customer payload"
//	@Success		201			{object}	domain.Customer
//	@Failure		400			{object}	map[string]string	"Validation error"
//	@Failure		409			{object}	map[string]string	"Email already in use"
//	@Failure		422			{object}	map[string]string	"Invalid tax ID"
//	@Router			/v1/customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var customer domain.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}
	if err := h.srv.CreateCustomer(c.Request.Context(), &customer); err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, customer)
}

// UpdateCustomer godoc
//
//	@Summary		Update a customer
//	@Description	Full update of a customer's fields. Status and version are not changed here — use PATCH /status for status transitions.
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string			true	"Customer UUID"
//	@Param			customer	body		domain.Customer	true	"Updated customer payload"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Router			/v1/customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UUID format"})
		return
	}
	var customer domain.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}
	customer.ID = id
	if err := h.srv.UpdateCustomer(c.Request.Context(), &customer); err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "customer updated successfully"})
}

// GetCustomerByID godoc
//
//	@Summary		Get a customer by ID
//	@Description	Returns a single customer by UUID.
//	@Tags			customers
//	@Produce		json
//	@Param			id	path		string	true	"Customer UUID"
//	@Success		200	{object}	domain.Customer
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/v1/customers/{id} [get]
func (h *CustomerHandler) GetCustomerByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UUID format"})
		return
	}
	customer, err := h.srv.SearchCustomer(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, customer)
}

// SearchByTaxID is an internal helper called by ListCustomers when ?taxId= is present.
// It is not a separate route and has no Swagger annotation to avoid duplicate route warnings.
func (h *CustomerHandler) SearchByTaxID(c *gin.Context) {
	taxID := c.Query("taxId")
	if taxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "taxId query param is required"})
		return
	}

	customer, err := h.srv.SearchByTaxID(c.Request.Context(), taxID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, customer)
}

// DeleteCustomer godoc
//
//	@Summary		Delete a customer
//	@Description	Soft-deletes a customer. Blocked customers cannot be deleted.
//	@Tags			customers
//	@Param			id	path	string	true	"Customer UUID"
//	@Success		204
//	@Failure		400	{object}	map[string]string
//	@Failure		403	{object}	map[string]string	"Customer is blocked"
//	@Failure		404	{object}	map[string]string
//	@Router			/v1/customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UUID format"})
		return
	}
	if err := h.srv.DeleteCustomer(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UpdateStatus godoc
//
//	@Summary		Update customer status
//	@Description	Transitions a customer to a new status. Valid transitions: pending→approved, pending→blocked, pending→terminated, approved→active, approved→blocked, approved→terminated, active→suspended, active→blocked, active→terminated, suspended→active, suspended→blocked, suspended→terminated, blocked→terminated.
//	@Tags			customers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Customer UUID"
//	@Param			body	body		map[string]string	true	"New status"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		422		{object}	map[string]string	"Invalid status transition"
//	@Router			/v1/customers/{id}/status [patch]
func (h *CustomerHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer ID"})
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	if err := h.srv.UpdateStatus(c.Request.Context(), id, body.Status); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated successfully"})
}

// ListCustomers godoc
//
//	@Summary		List customers
//	@Description	Returns a paginated list of customers. If ?taxId= is provided, delegates to tax ID search instead.
//	@Tags			customers
//	@Produce		json
//	@Param			taxId	query		string					false	"Search by tax ID (bypasses pagination)"
//	@Param			page	query		int						false	"Page number (default: 1)"
//	@Param			limit	query		int						false	"Items per page, max 100 (default: 20)"
//	@Param			status	query		string					false	"Filter by status (pending, approved, active, suspended, blocked, terminated)"
//	@Success		200		{object}	domain.PaginatedCustomers
//	@Failure		400		{object}	map[string]string
//	@Router			/v1/customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	if c.Query("taxId") != "" {
		h.SearchByTaxID(c)
		return
	}

	page := 1
	limit := 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page must be a positive integer"})
			return
		}
	}

	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer up to 100"})
			return
		}
	}

	params := domain.ListParams{
		Page:   page,
		Limit:  limit,
		Status: c.Query("status"),
	}

	result, err := h.srv.ListCustomers(c.Request.Context(), params)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
