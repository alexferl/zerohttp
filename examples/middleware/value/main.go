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
	userID, _ := middleware.GetContextValue[int](r, "userID")
	role, _ := middleware.GetContextValue[string](r, "role")

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"user_id": userID,
		"role":    role,
	})
}
