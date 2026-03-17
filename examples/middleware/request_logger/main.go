package main

import (
	"log"
	"net/http"
	"strings"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	zhlog "github.com/alexferl/zerohttp/log"
)

var apiKeyToTenant = map[string]string{
	"sk-1234567890abcdef": "tenant-acme",
	"sk-abcdef1234567890": "tenant-cyberdyne",
}

func main() {
	app := zh.New(config.Config{
		RequestLogger: config.RequestLoggerConfig{
			LogRequestBody:  true,
			LogResponseBody: true,
			MaxBodySize:     1024,
			Fields: []config.LogField{
				config.FieldMethod,
				config.FieldPath,
				config.FieldStatus,
				config.FieldDurationHuman,
				config.FieldRequestBody,
				config.FieldResponseBody,
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
