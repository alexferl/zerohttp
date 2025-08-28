package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	app.GET("/user", zh.HandlerFunc(userHandler),
		middleware.WithValue("userID", 123),
		middleware.WithValue("role", "admin"),
	)

	log.Fatal(app.Start())
}

func userHandler(w http.ResponseWriter, r *http.Request) error {
	userID, ok := middleware.GetContextValue[int](r, "userID")
	if !ok {
		return zh.R.JSON(w, 500, zh.M{"error": "userID not found"})
	}

	role, ok := middleware.GetContextValue[string](r, "role")
	if !ok {
		return zh.R.JSON(w, 500, zh.M{"error": "role not found"})
	}

	return zh.R.JSON(w, 200, zh.M{
		"user_id": userID,
		"role":    role,
		"message": "Got values from context using GetContextValue!",
	})
}
