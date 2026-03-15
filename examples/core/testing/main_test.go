package main

import (
	"net/http"
	"strings"
	"testing"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestHealthCheck(t *testing.T) {
	app := zh.New()
	app.GET("/health", zh.HandlerFunc(healthCheck))

	req := zhtest.NewRequest(http.MethodGet, "/health").
		WithHeader(zh.HeaderAccept, zh.MIMEApplicationJSON).
		Build()
	w := zhtest.Serve(app, req)

	zhtest.AssertWith(t, w).
		IsSuccess().
		JSONPathEqual("status", "ok")
}

func TestCreateUser(t *testing.T) {
	app := zh.New()
	app.POST("/users", zh.HandlerFunc(createUser))

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithHeader(zh.HeaderAccept, zh.MIMEApplicationJSON).
		WithJSON(map[string]string{"name": "Charlie", "email": "charlie@example.com"}).
		Build()
	w := zhtest.Serve(app, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusCreated).
		HeaderContains("Content-Type", "application/json").
		JSONPathEqual("name", "Charlie")
}

func TestCreateUserValidationError(t *testing.T) {
	app := zh.New()
	app.POST("/users", zh.HandlerFunc(createUser))

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithHeader(zh.HeaderAccept, zh.MIMEApplicationJSON).
		WithJSON(map[string]string{"name": "", "email": "invalid"}).
		Build()
	w := zhtest.Serve(app, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusUnprocessableEntity)
}

func TestGetUser(t *testing.T) {
	app := zh.New()
	app.GET("/users/{id}", zh.HandlerFunc(getUser))

	req := zhtest.NewRequest(http.MethodGet, "/users/1").
		WithHeader(zh.HeaderAccept, zh.MIMEApplicationJSON).
		Build()
	w := zhtest.Serve(app, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusOK).
		HeaderContains(zh.HeaderContentType, zh.MIMEApplicationJSON)

	body := w.Body.String()
	if !strings.Contains(body, "Alice") {
		t.Errorf("expected body to contain 'Alice', got %s", body)
	}
}

func TestGetUserNotFound(t *testing.T) {
	app := zh.New()
	app.GET("/users/{id}", zh.HandlerFunc(getUser))

	req := zhtest.NewRequest(http.MethodGet, "/users/999").
		WithHeader(zh.HeaderAccept, zh.MIMEApplicationJSON).
		Build()
	w := zhtest.Serve(app, req)

	zhtest.AssertWith(t, w).Status(http.StatusNotFound)
}
