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
	pp := pprof.New(app, pprof.DefaultConfig)
	log.Printf("pprof credentials - username: %s, password: %s", pp.Auth.Username, pp.Auth.Password)

	// Or with custom configuration:

	// Example 1: Custom prefix (auto-generates password)
	// cfg := pprof.DefaultConfig
	// cfg.Prefix = "/admin/pprof"
	// pp := pprof.New(app, cfg)
	// log.Printf("pprof credentials - username: %s, password: %s", pp.Auth.Username, pp.Auth.Password)

	// Example 2: With specific basic auth credentials
	// cfg := pprof.DefaultConfig
	// cfg.Auth = &pprof.AuthConfig{
	// 	Username: "admin",
	// 	Password: "secret",
	// }
	// pp := pprof.New(app, cfg)

	// Example 3: Disable authentication (not recommended for production)
	// cfg := pprof.DefaultConfig
	// cfg.Auth = &pprof.AuthConfig{} // empty = disabled
	// pprof.New(app, cfg)

	// Example 4: Selective endpoints (disable some profiles)
	// cfg := pprof.DefaultConfig
	// cfg.EnableBlock = false
	// cfg.EnableMutex = false
	// pp := pprof.New(app, cfg)

	log.Printf("pprof endpoints available at http://localhost:8080%s/", pp.Config.Prefix)
	log.Fatal(app.Start())
}
