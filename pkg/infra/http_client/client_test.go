package http_client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestGetWithHeadersAndAutoTraceID(t *testing.T) {
	var gotAuth, gotReqID, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotReqID = r.Header.Get(HeaderRequestID)
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "http_client_test")
	defer span.End()
	wantTraceID := span.SpanContext().TraceID().String()

	c := NewClient(Config{
		DefaultHeaders: map[string]string{"User-Agent": "go-infra-test"},
	})
	body, status, err := c.Get(ctx, srv.URL,
		WithHeader("Authorization", "Bearer token"),
	)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", body)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotUA != "go-infra-test" {
		t.Fatalf("User-Agent = %q", gotUA)
	}
	if gotReqID != wantTraceID {
		t.Fatalf("X-Request-ID = %q, want %q", gotReqID, wantTraceID)
	}
}

func TestExplicitRequestIDNotOverwritten(t *testing.T) {
	var gotReqID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReqID = r.Header.Get(HeaderRequestID)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "preserve")
	defer span.End()

	c := NewClient(Config{})
	_, _, err := c.Get(ctx, srv.URL,
		WithHeader(HeaderRequestID, "explicit-id"),
	)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotReqID != "explicit-id" {
		t.Fatalf("X-Request-ID = %q, want explicit-id (got span=%s)", gotReqID, span.SpanContext().TraceID())
	}
}

func TestNoTraceIDSkipsRequestID(t *testing.T) {
	var gotReqID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReqID = r.Header.Get(HeaderRequestID)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(Config{})
	_, _, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotReqID != "" {
		t.Fatalf("X-Request-ID should be empty without trace, got %q", gotReqID)
	}
}

func TestWithHeadersOverrideDefault(t *testing.T) {
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(Config{
		DefaultHeaders: map[string]string{"Content-Type": "text/plain"},
	})
	_, _, err := c.Post(context.Background(), srv.URL, []byte(`{}`), WithJSON())
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if gotCT != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotCT)
	}
}

func TestDoRequestCustomMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(Config{})
	_, _, err := c.DoRequest(context.Background(), "OPTIONS", srv.URL, nil)
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	if gotMethod != http.MethodOptions {
		t.Fatalf("method = %q", gotMethod)
	}
}

func TestNilClient(t *testing.T) {
	var c *Client
	_, _, err := c.Get(context.Background(), "http://example.com")
	if err != ErrNilClient {
		t.Fatalf("err = %v, want ErrNilClient", err)
	}
}

func TestResponseBodyTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789")) // 10 bytes
	}))
	defer srv.Close()

	c := NewClient(Config{MaxResponseBodyBytes: 4})
	body, status, err := c.Get(context.Background(), srv.URL)
	if err != ErrResponseBodyTooLarge {
		t.Fatalf("err = %v, want ErrResponseBodyTooLarge", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if body != nil {
		t.Fatalf("body should be nil on too-large, got %q", body)
	}
}

func TestMetricsRecordsTemplatePath(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	// Reset sync.Once so instruments bind to this test MeterProvider.
	clientMetricsOnce = sync.Once{}
	clientRequestsTotal = nil
	clientDurationSeconds = nil

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	c := NewClient(Config{MetricsEnabled: true})
	_, _, err := c.Get(context.Background(), srv.URL+"/users/1001/orders",
		WithMetricPath("/users/:id/orders"),
	)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	path, status := collectClientRequestLabels(t, reader)
	if path != "/users/:id/orders" {
		t.Fatalf("path label = %q, want /users/:id/orders", path)
	}
	if status != 200 {
		t.Fatalf("status label = %d, want 200", status)
	}
}

func TestMetricsFallsBackToRealPathWithoutDigits(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	clientMetricsOnce = sync.Once{}
	clientRequestsTotal = nil
	clientDurationSeconds = nil

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(Config{MetricsEnabled: true})
	// v1 is not a pure-digit segment, so the real path is kept.
	_, _, err := c.Get(context.Background(), srv.URL+"/api/v1/ping")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	path, _ := collectClientRequestLabels(t, reader)
	if path != "/api/v1/ping" {
		t.Fatalf("path label = %q, want /api/v1/ping", path)
	}
}

func TestMetricsUnknownWhenPathHasDigits(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	clientMetricsOnce = sync.Once{}
	clientRequestsTotal = nil
	clientDurationSeconds = nil

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(Config{MetricsEnabled: true})
	_, _, err := c.Get(context.Background(), srv.URL+"/users/1001/orders")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	path, _ := collectClientRequestLabels(t, reader)
	if path != metricPathUnknown {
		t.Fatalf("path label = %q, want %q", path, metricPathUnknown)
	}
}

func collectClientRequestLabels(t *testing.T, reader *sdkmetric.ManualReader) (path string, status int64) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http_client_requests_total" {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("unexpected data type %T", m.Data)
			}
			if len(sum.DataPoints) == 0 {
				t.Fatal("no data points")
			}
			for _, kv := range sum.DataPoints[0].Attributes.ToSlice() {
				switch string(kv.Key) {
				case "path":
					path = kv.Value.AsString()
				case "status":
					status = kv.Value.AsInt64()
				}
			}
			return path, status
		}
	}
	t.Fatal("http_client_requests_total not found")
	return "", 0
}

func TestMetricsDisabledNoOp(t *testing.T) {
	c := NewClient(Config{MetricsEnabled: false})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	_, _, err := c.Get(context.Background(), srv.URL, WithMetricPath("/ping"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
}
