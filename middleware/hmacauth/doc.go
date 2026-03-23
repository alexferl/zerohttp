// Package hmacauth provides HMAC request signing authentication.
//
// HMAC authentication verifies request integrity using a shared secret
// and a signature generated from the request content (similar to AWS Signature v4).
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/hmacauth"
//
//	app.Use(hmacauth.New(hmacauth.Config{
//	    Secrets: map[string]string{
//	        "key-id-1": "secret-for-key-1",
//	    },
//	}))
//
// # Signing Requests
//
// Clients sign requests using the SignRequest function:
//
//	sig := hmacauth.SignRequest(req, "key-id-1", "secret-for-key-1", nil)
//	req.Header.Set("Authorization", sig.String())
package hmacauth
