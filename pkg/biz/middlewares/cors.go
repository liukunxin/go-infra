package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"slices"
	"strings"
)

type CorsOptions struct {
	AllowOrigins     []string
	AllowHeaders     []string
	AllowMethods     []string
	ExposeHeaders    []string
	AllowCredentials bool
}

type CorsOption func(*CorsOptions)

func WithAllowOrigins(origins ...string) CorsOption {
	return func(o *CorsOptions) {
		o.AllowOrigins = normalizeValues(origins, normalizeOrigin)
	}
}

func WithAllowHeaders(headers ...string) CorsOption {
	return func(o *CorsOptions) {
		o.AllowHeaders = normalizeValues(headers, strings.TrimSpace)
	}
}

func WithAllowMethods(methods ...string) CorsOption {
	return func(o *CorsOptions) {
		o.AllowMethods = normalizeValues(methods, strings.TrimSpace)
	}
}

func WithExposeHeaders(headers ...string) CorsOption {
	return func(o *CorsOptions) {
		o.ExposeHeaders = normalizeValues(headers, strings.TrimSpace)
	}
}

func WithAllowCredentials(allow bool) CorsOption {
	return func(o *CorsOptions) {
		o.AllowCredentials = allow
	}
}

func CorsMiddleware(opts ...CorsOption) gin.HandlerFunc {
	options := defaultCorsOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return func(c *gin.Context) {
		origin := normalizeOrigin(c.Request.Header.Get("Origin"))
		if origin != "" && originAllowed(origin, options.AllowOrigins) {
			c.Header("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", boolToString(options.AllowCredentials))
			c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(options.AllowHeaders, ", "))
			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(options.AllowMethods, ", "))
			if len(options.ExposeHeaders) > 0 {
				c.Writer.Header().Set("Access-Control-Expose-Headers", strings.Join(options.ExposeHeaders, ", "))
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func defaultCorsOptions() CorsOptions {
	return CorsOptions{
		AllowHeaders:     []string{"*"},
		AllowMethods:     []string{"POST", "OPTIONS", "GET", "PUT", "PATCH", "DELETE"},
		AllowCredentials: true,
	}
}

func originAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return true
	}
	if slices.Contains(allowedOrigins, "*") {
		return true
	}
	return slices.Contains(allowedOrigins, origin)
}

func normalizeValues(values []string, transform func(string) string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		normalized := transform(v)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeOrigin(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
