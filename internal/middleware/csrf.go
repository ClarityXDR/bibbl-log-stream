package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CSRFConfig defines configuration for CSRF protection.
type CSRFConfig struct {
	// TokenLength is the length of the generated token (default: 32)
	TokenLength int

	// TokenLookup defines where to look for the token (default: "header:X-CSRF-Token")
	TokenLookup string

	// CookieName is the name of the CSRF cookie (default: "csrf_token")
	CookieName string

	// CookieSecure sets the Secure flag on the cookie (default: true)
	CookieSecure bool

	// CookieSameSite sets the SameSite attribute (default: "Strict")
	CookieSameSite string

	// Expiration is how long the token is valid (default: 24h)
	Expiration time.Duration

	// ExemptPaths are paths that don't require CSRF tokens (e.g., /api/v1/health)
	ExemptPaths []string
}

// CSRFMiddleware provides CSRF token generation and validation.
type CSRFMiddleware struct {
	config CSRFConfig
	tokens sync.Map // token -> expiry time
}

// NewCSRF creates a new CSRF middleware with the given config.
func NewCSRF(config CSRFConfig) *CSRFMiddleware {
	if config.TokenLength == 0 {
		config.TokenLength = 32
	}
	if config.TokenLookup == "" {
		config.TokenLookup = "header:X-CSRF-Token"
	}
	if config.CookieName == "" {
		config.CookieName = "csrf_token"
	}
	if config.CookieSameSite == "" {
		config.CookieSameSite = "Strict"
	}
	if config.Expiration == 0 {
		config.Expiration = 24 * time.Hour
	}

	csrf := &CSRFMiddleware{
		config: config,
	}

	// Cleanup goroutine for expired tokens
	go csrf.cleanup()

	return csrf
}

// Handler returns a Fiber middleware handler.
func (c *CSRFMiddleware) Handler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// Skip exempt paths
		path := ctx.Path()
		for _, exempt := range c.config.ExemptPaths {
			if path == exempt {
				return ctx.Next()
			}
		}

		// GET, HEAD, OPTIONS don't need CSRF validation
		method := ctx.Method()
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			// Generate token if not present
			if ctx.Cookies(c.config.CookieName) == "" {
				token := c.generateToken()
				c.setTokenCookie(ctx, token)
			}
			return ctx.Next()
		}

		// State-changing methods require validation
		cookieToken := ctx.Cookies(c.config.CookieName)
		if cookieToken == "" {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token missing",
			})
		}

		// Extract token from header
		headerToken := ctx.Get("X-CSRF-Token")
		if headerToken == "" {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token required",
			})
		}

		// Validate token
		if !c.validateToken(cookieToken, headerToken) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token invalid",
			})
		}

		return ctx.Next()
	}
}

// generateToken creates a new CSRF token.
func (c *CSRFMiddleware) generateToken() string {
	b := make([]byte, c.config.TokenLength)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based (not ideal but better than failing)
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}

	token := base64.URLEncoding.EncodeToString(b)

	// Store token with expiration
	c.tokens.Store(token, time.Now().Add(c.config.Expiration))

	return token
}

// validateToken checks if the provided tokens match and are valid.
func (c *CSRFMiddleware) validateToken(cookieToken, headerToken string) bool {
	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
		return false
	}

	// Check if token exists and is not expired
	expiryRaw, ok := c.tokens.Load(cookieToken)
	if !ok {
		return false
	}

	expiry, ok := expiryRaw.(time.Time)
	if !ok || time.Now().After(expiry) {
		c.tokens.Delete(cookieToken)
		return false
	}

	return true
}

// setTokenCookie sets the CSRF token cookie.
func (c *CSRFMiddleware) setTokenCookie(ctx *fiber.Ctx, token string) {
	ctx.Cookie(&fiber.Cookie{
		Name:     c.config.CookieName,
		Value:    token,
		Expires:  time.Now().Add(c.config.Expiration),
		HTTPOnly: true,
		Secure:   c.config.CookieSecure,
		SameSite: c.config.CookieSameSite,
		Path:     "/",
	})
}

// cleanup removes expired tokens periodically.
func (c *CSRFMiddleware) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.tokens.Range(func(key, value interface{}) bool {
			if expiry, ok := value.(time.Time); ok && now.After(expiry) {
				c.tokens.Delete(key)
			}
			return true
		})
	}
}
