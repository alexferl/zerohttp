package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	zlog "github.com/alexferl/zerohttp/log"
)

func main() {
	// Create a logger and set the log level
	// Available levels: DebugLevel, InfoLevel (default), WarnLevel, ErrorLevel, PanicLevel, FatalLevel
	logger := zlog.NewDefaultLogger()
	logger.SetLevel(zlog.DebugLevel)

	app := zh.New(config.Config{
		Logger: logger,
	})

	app.GET("/demo", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// These will all be logged since level is DebugLevel
		app.Logger().Debug("This is a debug message")
		app.Logger().Info("This is an info message")
		app.Logger().Warn("This is a warning message")
		app.Logger().Error("This is an error message")

		return zh.Render.JSON(w, http.StatusOK, zh.M{
			"message": "Check your console for log output",
		})
	}))

	app.GET("/filter", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Change log level to Warn - only Warn and above will show
		app.Logger().(*zlog.DefaultLogger).SetLevel(zlog.WarnLevel)

		app.Logger().Debug("This debug message will NOT be logged")
		app.Logger().Info("This info message will NOT be logged")
		app.Logger().Warn("This warning message WILL be logged")
		app.Logger().Error("This error message WILL be logged")

		// Reset back to Info for other requests
		app.Logger().(*zlog.DefaultLogger).SetLevel(zlog.InfoLevel)

		return zh.Render.JSON(w, http.StatusOK, zh.M{
			"message": "Check your console - only Warn and Error should appear",
		})
	}))

	log.Fatal(app.Start())
}
