package main

import (
	"fmt"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
)

type BulkCreateRequest struct {
	Tags       []string       `json:"tags" validate:"each,min=3,max=20,alphanum"`
	Recipients []string       `json:"recipients" validate:"each,email"`
	Products   []ProductInput `json:"products" validate:"required,min=1,each"`
}

type ProductInput struct {
	SKU   string  `json:"sku" validate:"required,uppercase,len=10"`
	Name  string  `json:"name" validate:"required,min=2,max=100"`
	Price float64 `json:"price" validate:"gt=0"`
}

type CreateOrganizationRequest struct {
	Name    string      `json:"name" validate:"required,min=2,max=100"`
	Slug    string      `json:"slug" validate:"required,lowercase,min=3,max=50"`
	Owner   UserInfo    `json:"owner" validate:"required"`
	Admins  []UserInfo  `json:"admins" validate:"each"`
	Billing BillingInfo `json:"billing" validate:"required"`
}

type UserInfo struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin member viewer"`
}

type BillingInfo struct {
	Plan      string `json:"plan" validate:"required,oneof=free starter pro enterprise"`
	CardToken string `json:"card_token"`
}

func (b BillingInfo) Validate() error {
	if b.Plan != "free" && b.CardToken == "" {
		return fmt.Errorf("card_token is required for paid plans")
	}
	return nil
}

func main() {
	app := zh.New()

	app.POST("/bulk", zh.HandlerFunc(bulkCreateHandler))
	app.POST("/organizations", zh.HandlerFunc(createOrganizationHandler))

	log.Fatal(app.Start())
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
