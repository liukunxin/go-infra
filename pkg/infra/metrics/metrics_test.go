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

	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mw := httptest.NewRecorder()
	router.ServeHTTP(mw, mreq)
	body, _ := io.ReadAll(mw.Body)
	text := string(body)

	if !strings.Contains(text, `app_name="my-demo-app"`) {
		t.Fatalf("metrics output missing app_name label, got:\n%s", text)
	}
}
