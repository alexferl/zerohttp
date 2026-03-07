package main

import (
	"log"
	"net/http"
	"os"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/rs/zerolog"
)

func main() {
	zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Caller().
		Logger()
	logger := NewZerologAdapter(zl)

	app := zh.New(config.WithLogger(logger))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		app.Logger().Info("I'm a log!")
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	log.Fatal(app.Start())
}
