package main

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	zh "github.com/alexferl/zerohttp"
)

func init() {
	// Register custom validators at startup
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

type CreateOrderRequest struct {
	CustomerID string      `json:"customer_id" validate:"required,uuid"`
	Items      []OrderItem `json:"items" validate:"required,min=1,max=100"`
	Status     string      `json:"status" validate:"required,oneof=pending paid shipped"`
	Discount   float64     `json:"discount" validate:"gte=0"`
	Total      float64     `json:"total" validate:"gte=0"`
}

func (r CreateOrderRequest) Validate() error {
	var calculatedTotal float64
	for _, item := range r.Items {
		calculatedTotal += item.Price * float64(item.Quantity)
	}
	if r.Total != calculatedTotal {
		return fmt.Errorf("total must equal sum of items (expected %.2f, got %.2f)", calculatedTotal, r.Total)
	}
	if r.Discount > r.Total {
		return fmt.Errorf("discount cannot exceed total")
	}
	if r.Status == "shipped" && r.Discount > 0 {
		return fmt.Errorf("cannot apply discount to shipped orders")
	}
	return nil
}

type OrderItem struct {
	ProductID string  `json:"product_id" validate:"required,uuid"`
	Quantity  int     `json:"quantity" validate:"required,min=1,max=999"`
	Price     float64 `json:"price" validate:"required,gte=0"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,strong_password"`
	Confirm  string `json:"confirm_password" validate:"required"`
}

func (r RegisterRequest) Validate() error {
	if r.Password != r.Confirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func main() {
	app := zh.New()

	app.POST("/orders", zh.HandlerFunc(createOrderHandler))
	app.POST("/register", zh.HandlerFunc(registerHandler))

	log.Fatal(app.Start())
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) error {
	var req CreateOrderRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message":     "Order created",
		"customer_id": req.CustomerID,
		"status":      req.Status,
		"item_count":  len(req.Items),
		"total":       req.Total - req.Discount,
	})
}

func registerHandler(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	return zh.R.JSON(w, http.StatusCreated, zh.M{
		"message": "Registration successful",
		"email":   req.Email,
	})
}
