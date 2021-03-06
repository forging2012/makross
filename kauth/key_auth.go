package kauth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/insionng/makross"
	"github.com/insionng/makross/skipper"
)

type (
	// KeyAuthConfig defines the config for KeyAuth middleware.
	KeyAuthConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper skipper.Skipper

		// KeyLookup is a string in the form of "<source>:<name>" that is used
		// to extract key from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		KeyLookup string `json:"key_lookup"`

		// AuthScheme to be used in the Authorization header.
		// Optional. Default value "Bearer".
		AuthScheme string

		// Validator is a function to validate key.
		// Required.
		Validator KeyAuthValidator
	}

	// KeyAuthValidator defines a function to validate KeyAuth credentials.
	KeyAuthValidator func(string, *makross.Context) (error, bool)

	keyExtractor func(*makross.Context) (string, error)
)

var (
	// DefaultKeyAuthConfig is the default KeyAuth middleware config.
	DefaultKeyAuthConfig = KeyAuthConfig{
		Skipper:    skipper.DefaultSkipper,
		KeyLookup:  "header:" + makross.HeaderAuthorization,
		AuthScheme: "Bearer",
	}
)

// KeyAuth returns an KeyAuth middleware.
//
// For valid key it calls the next handler.
// For invalid key, it sends "401 - Unauthorized" response.
// For missing key, it sends "400 - Bad Request" response.
func KeyAuth(fn KeyAuthValidator) makross.Handler {
	c := DefaultKeyAuthConfig
	c.Validator = fn
	return KeyAuthWithConfig(c)
}

// KeyAuthWithConfig returns an KeyAuth middleware with config.
// See `KeyAuth()`.
func KeyAuthWithConfig(config KeyAuthConfig) makross.Handler {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultKeyAuthConfig.Skipper
	}
	// Defaults
	if config.AuthScheme == "" {
		config.AuthScheme = DefaultKeyAuthConfig.AuthScheme
	}
	if config.KeyLookup == "" {
		config.KeyLookup = DefaultKeyAuthConfig.KeyLookup
	}
	if config.Validator == nil {
		panic("key-auth middleware requires a validator function")
	}

	// Initialize
	parts := strings.Split(config.KeyLookup, ":")
	extractor := keyFromHeader(parts[1], config.AuthScheme)
	switch parts[0] {
	case "query":
		extractor = keyFromQuery(parts[1])
	}

	return func(c *makross.Context) error {
		if config.Skipper(c) {
			return c.Next()
		}

		// Extract and verify key
		key, err := extractor(c)
		if err != nil {
			return makross.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		err, valid := config.Validator(key, c)
		if err != nil {
			return err
		} else if valid {
			return c.Next()
		}

		return makross.ErrUnauthorized
	}

}

// keyFromHeader returns a `keyExtractor` that extracts key from the request header.
func keyFromHeader(header string, authScheme string) keyExtractor {
	return func(c *makross.Context) (string, error) {
		auth := c.Request.Header.Get(header)
		if auth == "" {
			return "", errors.New("Missing key in request header")
		}
		if header == makross.HeaderAuthorization {
			l := len(authScheme)
			if len(auth) > l+1 && auth[:l] == authScheme {
				return auth[l+1:], nil
			}
			return "", errors.New("Invalid key in the request header")
		}
		return auth, nil
	}
}

// keyFromQuery returns a `keyExtractor` that extracts key from the query string.
func keyFromQuery(param string) keyExtractor {
	return func(c *makross.Context) (string, error) {
		key := c.Query(param)
		if key == "" {
			return "", errors.New("Missing key in the query string")
		}
		return key, nil
	}
}
