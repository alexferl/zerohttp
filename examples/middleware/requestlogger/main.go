package main

import (
	"log"
	"net/http"
	"strings"

	zh "github.com/alexferl/zerohttp"
	zhlog "github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/middleware/requestlogger"
)

var apiKeyToTenant = map[string]string{
	"sk-1234567890abcdef": "tenant-acme",
	"sk-abcdef1234567890": "tenant-cyberdyne",
}

func main() {
	app := zh.New(zh.Config{
		RequestLogger: requestlogger.Config{
			LogRequestBody:  true,
			LogResponseBody: true,
			MaxBodySize:     1024,
			Fields: []requestlogger.LogField{
				requestlogger.FieldMethod,
				requestlogger.FieldPath,
				requestlogger.FieldStatus,
				requestlogger.FieldDurationHuman,
				requestlogger.FieldRequestBody,
				requestlogger.FieldResponseBody,
			},
			CustomFields: func(r *http.Request) []zhlog.Field {
				var fields []zhlog.Field

				if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
					if tenantID, ok := apiKeyToTenant[apiKey]; ok {
						fields = append(fields, zhlog.F("tenant_id", tenantID))
					}
				}

				if strings.HasPrefix(r.URL.Path, "/admin/") {
					fields = append(fields, zhlog.F("access_level", "admin"))
				}

				return fields
			},
		},
	})

	app.POST("/api/login", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status":  "success",
			"message": "Login successful",
			"token":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		})
	}))

	app.GET("/admin/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"users": []string{"user1", "user2", "user3"},
		})
	}))

	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"status": "healthy",
		})
	}))

	log.Fatal(app.Start())
}
