package main

import (
	"log"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/pprof"
)

func main() {
	app := zh.New()

	// Basic usage with defaults - auto-generates secure password
	// Credentials are available via the returned PProf struct
	pp := pprof.New(app)
	log.Printf("pprof credentials - username: %s, password: %s", pp.Auth.Username, pp.Auth.Password)
	log.Printf("pprof endpoints available at http://localhost:8080%s/", pp.Config.Prefix)

	log.Fatal(app.Start())
}
