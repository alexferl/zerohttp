package main

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	zh "github.com/alexferl/zerohttp"
)

func init() {
	// Register custom validators at startup
	registerCustomValidators()
}

// ============================================
// Basic Validation Examples
// ============================================

// CreateUserRequest demonstrates basic validation
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min=13,max=120"`
	Username string `json:"username" validate:"required,alphanum,min=3,max=20"`
}

// UpdateUserRequest demonstrates optional fields with omitempty
type UpdateUserRequest struct {
	Name  *string `json:"name" validate:"omitempty,min=2,max=50"`
	Email *string `json:"email" validate:"omitempty,email"`
	Age   *int    `json:"age" validate:"omitempty,min=13,max=120"`
}

// ============================================
// Advanced Validation with Custom Validate() Method
// ============================================

// CreateOrderRequest demonstrates cross-field validation via Validate() method
type CreateOrderRequest struct {
	CustomerID string      `json:"customer_id" validate:"required,uuid"`
	Items      []OrderItem `json:"items" validate:"required,min=1,max=100"`
	Status     string      `json:"status" validate:"required,oneof=pending paid shipped"`
	Discount   float64     `json:"discount" validate:"gte=0"`
	Total      float64     `json:"total" validate:"gte=0"`
}

// Validate implements custom cross-field validation
func (r CreateOrderRequest) Validate() error {
	// Calculate expected total from items
	var calculatedTotal float64
	for _, item := range r.Items {
		calculatedTotal += item.Price * float64(item.Quantity)
	}

	// Validate total matches sum of items
	if r.Total != calculatedTotal {
		return fmt.Errorf("total must equal sum of items (expected %.2f, got %.2f)", calculatedTotal, r.Total)
	}

	// Validate discount doesn't exceed total
	if r.Discount > r.Total {
		return fmt.Errorf("discount cannot exceed total")
	}

	// Validate status transitions (business logic)
	if r.Status == "shipped" && r.Discount > 0 {
		return fmt.Errorf("cannot apply discount to shipped orders")
	}

	return nil
}

// OrderItem is validated as part of the nested slice
type OrderItem struct {
	ProductID string  `json:"product_id" validate:"required,uuid"`
	Quantity  int     `json:"quantity" validate:"required,min=1,max=999"`
	Price     float64 `json:"price" validate:"required,gte=0"`
}

// ============================================
// Custom Validator Registration Example
// ============================================

// RegisterRequest uses a custom validator
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,strong_password"`
	Confirm  string `json:"confirm_password" validate:"required"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
	Referral string `json:"referral_code" validate:"omitempty,uppercase,len=8"`
}

// Validate ensures password and confirmation match
func (r RegisterRequest) Validate() error {
	if r.Password != r.Confirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

// ============================================
// Each Validator Example (Slice Element Validation)
// ============================================

// BulkCreateRequest uses each for validating slice elements
type BulkCreateRequest struct {
	// Each tag must be 3-20 chars, alphanumeric only
	Tags []string `json:"tags" validate:"each,min=3,max=20,alphanum"`

	// Each email must be valid
	Recipients []string `json:"recipients" validate:"each,email"`

	// Validate each item in a bulk insert
	Products []ProductInput `json:"products" validate:"required,min=1,each"`
}

// ProductInput for bulk operations
type ProductInput struct {
	SKU         string  `json:"sku" validate:"required,uppercase,len=10"`
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Price       float64 `json:"price" validate:"gt=0"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

// ============================================
// Complex Nested Struct Example
// ============================================

// CreateOrganizationRequest shows deep nesting
type CreateOrganizationRequest struct {
	Name    string      `json:"name" validate:"required,min=2,max=100"`
	Slug    string      `json:"slug" validate:"required,lowercase,min=3,max=50"`
	Owner   UserInfo    `json:"owner" validate:"required"`
	Admins  []UserInfo  `json:"admins" validate:"each"`
	Billing BillingInfo `json:"billing" validate:"required"`
}

// UserInfo nested struct
type UserInfo struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin member viewer"`
}

// BillingInfo with embedded validation
type BillingInfo struct {
	Plan       string  `json:"plan" validate:"required,oneof=free starter pro enterprise"`
	CardToken  string  `json:"card_token"`
	CouponCode *string `json:"coupon_code" validate:"omitempty,uppercase,len=10"`
}

// Validate implements conditional validation for billing
func (b BillingInfo) Validate() error {
	// Require card token for paid plans
	if b.Plan != "free" && b.CardToken == "" {
		return fmt.Errorf("card_token is required for paid plans")
	}
	return nil
}

// ============================================
// Response Validation Example
// ============================================

// UserResponse is validated before sending to ensure data integrity
type UserResponse struct {
	ID        string    `json:"id" validate:"required,uuid"`
	Email     string    `json:"email" validate:"required,email"`
	Name      string    `json:"name" validate:"required,min=1,max=100"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Validate ensures response data consistency
func (u UserResponse) Validate() error {
	if u.UpdatedAt.Before(u.CreatedAt) {
		return fmt.Errorf("updated_at cannot be before created_at")
	}
	return nil
}

func main() {
	app := zh.New()

	// Basic routes
	app.POST("/users", zh.HandlerFunc(createUserHandler))
	app.PATCH("/users/{id}", zh.HandlerFunc(updateUserHandler))

	// Advanced routes
	app.POST("/orders", zh.HandlerFunc(createOrderHandler))
	app.POST("/register", zh.HandlerFunc(registerHandler))
	app.POST("/bulk", zh.HandlerFunc(bulkCreateHandler))
	app.POST("/organizations", zh.HandlerFunc(createOrganizationHandler))

	// Response validation demo
	app.GET("/users/{id}", zh.HandlerFunc(getUserHandler))

	log.Println("Server starting on :8080")
	log.Println("")
	log.Println("=== Basic Examples ===")
	log.Println("  POST /users           - Create user with validation")
	log.Println("  PATCH /users/123      - Update user (optional fields)")
	log.Println("")
	log.Println("=== Advanced Examples ===")
	log.Println("  POST /orders          - Cross-field validation (total/discount/items)")
	log.Println("  POST /register        - Custom validators + password confirmation")
	log.Println("  POST /bulk            - Dive validator for slice elements")
	log.Println("  POST /organizations   - Deep nested struct validation")
	log.Println("  GET  /users/123       - Response validation demo")
	log.Println("")
	log.Println("=== Try These Commands ===")
	printExamples()
	log.Fatal(app.Start())
}

func printExamples() {
	// Basic user creation
	log.Println(`
# Create user (valid)
curl -X POST http://localhost:8080/users -H "Content-Type: application/json" -d '{
  "name":"John Doe",
  "email":"john@example.com",
  "age":25,
  "username":"johndoe"
}'

# Create user (validation errors)
curl -X POST http://localhost:8080/users -H "Content-Type: application/json" -d '{
  "name":"J",
  "email":"bad-email",
  "age":5,
  "username":"ab"
}'`)

	// Order with cross-field validation
	log.Println(`
# Create order (valid - total matches items)
curl -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d '{
  "customer_id":"550e8400-e29b-41d4-a716-446655440000",
  "status":"pending",
  "discount":5.00,
  "total":59.98,
  "items":[
    {"product_id":"550e8400-e29b-41d4-a716-446655440001","quantity":2,"price":29.99}
  ]
}'

# Create order (fails cross-field validation - total mismatch)
curl -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d '{
  "customer_id":"550e8400-e29b-41d4-a716-446655440000",
  "status":"pending",
  "total":100.00,
  "items":[
    {"product_id":"550e8400-e29b-41d4-a716-446655440001","quantity":1,"price":29.99}
  ]
}'`)

	// Custom validator
	log.Println(`
# Register with custom password validator
curl -X POST http://localhost:8080/register -H "Content-Type: application/json" -d '{
  "email":"user@example.com",
  "password":"StrongP@ss123",
  "confirm_password":"StrongP@ss123",
  "phone":"+14155552671",
  "referral_code":"ABC12345"
}'

# Register (weak password fails custom validator)
curl -X POST http://localhost:8080/register -H "Content-Type: application/json" -d '{
  "email":"user@example.com",
  "password":"123",
  "confirm_password":"123"
}'`)

	// Each validator
	log.Println(`
# Bulk create with each validation
curl -X POST http://localhost:8080/bulk -H "Content-Type: application/json" -d '{
  "tags":["golang","validation","api"],
  "recipients":["admin@example.com","user@example.com"],
  "products":[
    {"sku":"PROD123456","name":"Product One","price":29.99},
    {"sku":"PROD789012","name":"Product Two","price":49.99}
  ]
}'

# Bulk create (fails each validation on tags)
curl -X POST http://localhost:8080/bulk -H "Content-Type: application/json" -d '{
  "tags":["a","way-too-long-tag-name-here"],
  "products":[{"sku":"prod123","name":"X","price":0}]
}'`)

	// Nested structs
	log.Println(`
# Create organization (deep nesting)
curl -X POST http://localhost:8080/organizations -H "Content-Type: application/json" -d '{
  "name":"Acme Corp",
  "slug":"acme-corp",
  "owner":{"name":"John","email":"john@example.com","role":"admin"},
  "admins":[{"name":"Jane","email":"jane@example.com","role":"admin"}],
  "billing":{"plan":"pro","card_token":"tok_visa"}
}'

# Create organization (fails conditional billing validation)
curl -X POST http://localhost:8080/organizations -H "Content-Type: application/json" -d '{
  "name":"Acme Corp",
  "slug":"acme-corp",
  "owner":{"name":"John","email":"john@example.com","role":"admin"},
  "billing":{"plan":"pro"}
}'`)
}

// ============================================
// Custom Validator Registration
// ============================================

func registerCustomValidators() {
	// Register strong_password validator
	zh.V.Register("strong_password", func(value reflect.Value, param string) error {
		if value.Kind() != reflect.String {
			return fmt.Errorf("strong_password only supports strings")
		}
		password := value.String()

		if len(password) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}
		if !strings.ContainsAny(password, "0123456789") {
			return fmt.Errorf("password must contain at least one digit")
		}
		if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			return fmt.Errorf("password must contain at least one uppercase letter")
		}
		if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
			return fmt.Errorf("password must contain at least one lowercase letter")
		}
		if !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
			return fmt.Errorf("password must contain at least one special character")
		}
		return nil
	})
}

// ============================================
// Handler Functions
// ============================================

func createUserHandler(w http.ResponseWriter, r *http.Request) error {
	var req CreateUserRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message":  "User created",
		"name":     req.Name,
		"email":    req.Email,
		"age":      req.Age,
		"username": req.Username,
	})
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) error {
	var req UpdateUserRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	response := zh.M{"message": "User updated"}
	if req.Name != nil {
		response["name"] = *req.Name
	}
	if req.Email != nil {
		response["email"] = *req.Email
	}
	if req.Age != nil {
		response["age"] = *req.Age
	}

	return zh.R.JSON(w, http.StatusOK, response)
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) error {
	var req CreateOrderRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	// Calculate final total after discount
	finalTotal := req.Total - req.Discount

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message":     "Order created",
		"customer_id": req.CustomerID,
		"status":      req.Status,
		"item_count":  len(req.Items),
		"subtotal":    req.Total,
		"discount":    req.Discount,
		"total":       finalTotal,
	})
}

func registerHandler(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message":       "Registration successful",
		"email":         req.Email,
		"phone":         req.Phone,
		"referral_code": req.Referral,
	})
}

func bulkCreateHandler(w http.ResponseWriter, r *http.Request) error {
	var req BulkCreateRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message":        "Bulk operation completed",
		"tags_count":     len(req.Tags),
		"products_count": len(req.Products),
	})
}

func createOrganizationHandler(w http.ResponseWriter, r *http.Request) error {
	var req CreateOrganizationRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message": "Organization created",
		"name":    req.Name,
		"slug":    req.Slug,
		"owner":   req.Owner.Email,
		"plan":    req.Billing.Plan,
	})
}

func getUserHandler(w http.ResponseWriter, r *http.Request) error {
	// Simulate fetching user data
	user := UserResponse{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		Email:     "user@example.com",
		Name:      "John Doe",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	// Use RenderAndValidate to validate response before sending
	// This catches server-side bugs (e.g., missing required fields)
	return zh.RenderAndValidate(w, http.StatusOK, user)
}
