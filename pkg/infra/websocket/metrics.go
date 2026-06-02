package websocket

import (
	"context"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	wsMetricsOnce sync.Once

	wsConnections      metric.Int64UpDownCounter
	wsMessagesTotal    metric.Int64Counter
	wsMessageBytes     metric.Int64Counter
	wsErrorsTotal      metric.Int64Counter
	wsReconnectsTotal  metric.Int64Counter
)

func ensureMetrics() {
	wsMetricsOnce.Do(func() {
		meter := otel.Meter("websocket")
		wsConnections, _ = meter.Int64UpDownCounter("ws_connections")
		wsMessagesTotal, _ = meter.Int64Counter("ws_messages_total")
		wsMessageBytes, _ = meter.Int64Counter("ws_message_bytes_total")
		wsErrorsTotal, _ = meter.Int64Counter("ws_errors_total")
		wsReconnectsTotal, _ = meter.Int64Counter("ws_reconnect_total")
	})
}

func recordConnectionDelta(ctx context.Context, delta int64, role string) {
	ensureMetrics()
	wsConnections.Add(ctx, delta, metric.WithAttributes(attribute.String("role", role)))
}

func recordMessage(ctx context.Context, role, direction string, msgType int, size int) {
	ensureMetrics()
	attrs := []attribute.KeyValue{
		attribute.String("role", role),
		attribute.String("direction", direction),
		attribute.String("type", strconv.Itoa(msgType)),
	}
	wsMessagesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	wsMessageBytes.Add(ctx, int64(size), metric.WithAttributes(attrs...))
}

func recordError(ctx context.Context, role string) {
	ensureMetrics()
	wsErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("role", role)))
}

func recordReconnect(ctx context.Context) {
	ensureMetrics()
	wsReconnectsTotal.Add(ctx, 1)
}
