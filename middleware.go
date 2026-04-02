package zerohttp

import (
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/middleware/recover"
	"github.com/alexferl/zerohttp/middleware/requestbodysize"
	"github.com/alexferl/zerohttp/middleware/requestid"
	"github.com/alexferl/zerohttp/middleware/requestlogger"
	"github.com/alexferl/zerohttp/middleware/securityheaders"
)

// DefaultMiddlewares returns the default set of middlewares with the provided configuration.
// The returned middlewares are applied in the following order:
//   - RequestID: Assigns a unique request ID to each request
//   - Recover: Recovers from panics and logs errors
//   - RequestBodySize: Limits the maximum request body size
//   - SecurityHeaders: Adds security-related HTTP headers
//   - RequestLogger: Logs HTTP requests and responses
func DefaultMiddlewares(cfg Config, logger log.Logger) []MiddlewareFunc {
	// Sync RequestID header configuration with Recover config
	recoverConfig := cfg.Recover
	recoverConfig.RequestIDHeader = cfg.RequestID.Header

	return []MiddlewareFunc{
		requestid.New(cfg.RequestID),
		recover.New(logger, recoverConfig),
		requestbodysize.New(cfg.RequestBodySize),
		securityheaders.New(cfg.SecurityHeaders),
		requestlogger.New(logger, cfg.RequestLogger),
	}
}
