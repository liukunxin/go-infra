package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHTTPSkipPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	_ = Shutdown(context.Background())
	if err := Init("skip-test-app"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = Shutdown(context.Background()) })
	if err := initHTTPMetrics(); err != nil {
		t.Fatalf("initHTTPMetrics: %v", err)
	}
	RegisterGinRoutes(router)

	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	body := scrapeMetricsBody(t, router)

	if strings.Contains(body, `path="/health"`) || strings.Contains(body, `path="/metrics"`) {
		t.Fatalf("skipped paths should not appear in metrics, got:\n%s", body)
	}
	if !strings.Contains(body, `path="/api/ping"`) {
		t.Fatalf("metrics output missing /api/ping, got:\n%s", body)
	}
}

func TestWithHTTPSkipPathsOption(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	_ = Shutdown(context.Background())
	if err := Init("skip-opt-app", WithHTTPSkipPaths("/readyz")); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = Shutdown(context.Background()) })
	if err := initHTTPMetrics(); err != nil {
		t.Fatalf("initHTTPMetrics: %v", err)
	}
	RegisterGinRoutes(router)

	router.GET("/readyz", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/readyz", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/ping", nil))

	body := scrapeMetricsBody(t, router)

	if strings.Contains(body, `path="/readyz"`) {
		t.Fatalf("WithHTTPSkipPaths path should not appear in metrics, got:\n%s", body)
	}
	if !strings.Contains(body, `path="/api/ping"`) {
		t.Fatalf("metrics output missing /api/ping, got:\n%s", body)
	}
}

func TestRegisterGinRoutesSkipPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	_ = Shutdown(context.Background())
	if err := Init("skip-reg-app"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = Shutdown(context.Background()) })
	if err := initHTTPMetrics(); err != nil {
		t.Fatalf("initHTTPMetrics: %v", err)
	}
	RegisterGinRoutes(router, "/livez")

	router.GET("/livez", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/livez", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/ping", nil))

	body := scrapeMetricsBody(t, router)

	if strings.Contains(body, `path="/livez"`) {
		t.Fatalf("RegisterGinRoutes skip path should not appear in metrics, got:\n%s", body)
	}
	if !strings.Contains(body, `path="/api/ping"`) {
		t.Fatalf("metrics output missing /api/ping, got:\n%s", body)
	}
}

func scrapeMetricsBody(t *testing.T, router *gin.Engine) string {
	t.Helper()
	mw := httptest.NewRecorder()
	router.ServeHTTP(mw, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	text, err := io.ReadAll(mw.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	return string(text)
}

func TestInitMetricsExposesAppNameLabel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	_ = Shutdown(context.Background())

	if err := Init("my-demo-app"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = Shutdown(context.Background()) })

	if err := initHTTPMetrics(); err != nil {
		t.Fatalf("initHTTPMetrics: %v", err)
	}
	RegisterGinRoutes(router)

	router.GET("/api/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	text := scrapeMetricsBody(t, router)

	if !strings.Contains(text, `app_name="my-demo-app"`) {
		t.Fatalf("metrics output missing app_name label, got:\n%s", text)
	}
}
