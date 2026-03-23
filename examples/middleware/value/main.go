package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/value"
)

func main() {
	app := zh.New()

	app.GET("/user", zh.HandlerFunc(userHandler),
		value.With("userID", 123),
		value.With("role", "admin"),
	)

	log.Fatal(app.Start())
}

func userHandler(w http.ResponseWriter, r *http.Request) error {
	userID, _ := value.Get[int](r, "userID")
	role, _ := value.Get[string](r, "role")

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"user_id": userID,
		"role":    role,
	})
}
