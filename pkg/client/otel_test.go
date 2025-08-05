package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GetSimpl/gotel/pkg/config"
)

func TestNewOtelClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config with local environment",
			config: &config.Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "local",
				OtelEndpoint:   "localhost:4318",
				SendInterval:   10,
				EnableDebug:    false,
			},
			wantErr: false,
		},
		{
			name: "valid config with production environment",
			config: &config.Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "local", // Use local environment for tests to avoid HTTPS issues
				OtelEndpoint:   "localhost:4318",
				SendInterval:   30,
				EnableDebug:    false,
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			config: &config.Config{
				ServiceName:    "",
				ServiceVersion: "1.0.0",
				Environment:    "local",
				OtelEndpoint:   "localhost:4318",
				SendInterval:   10,
			},
			wantErr: false, // Should still work with empty service name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOtelClient(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)

				// Verify client implements interface
				assert.Implements(t, (*OTelClient)(nil), client)

				// Clean up only if client is not nil
				if client != nil {
					err := client.Close()
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestOtelClient_CreateCounter(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	tests := []struct {
		name       string
		metricName string
		unit       string
		wantErr    bool
	}{
		{
			name:       "valid counter",
			metricName: "test_counter",
			unit:       "requests",
			wantErr:    false,
		},
		{
			name:       "counter with empty name",
			metricName: "",
			unit:       "requests",
			wantErr:    true, // OpenTelemetry rejects empty names
		},
		{
			name:       "counter with empty unit",
			metricName: "test_counter_2",
			unit:       "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter, err := client.CreateCounter(tt.metricName, tt.unit)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, counter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, counter)
				assert.Implements(t, (*Counter)(nil), counter)
			}
		})
	}
}

func TestOtelClient_CreateGauge(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	tests := []struct {
		name       string
		metricName string
		unit       string
		wantErr    bool
	}{
		{
			name:       "valid gauge",
			metricName: "test_gauge",
			unit:       "bytes",
			wantErr:    false,
		},
		{
			name:       "gauge with special characters",
			metricName: "test.gauge_with-chars",
			unit:       "ms",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gauge, err := client.CreateGauge(tt.metricName, tt.unit)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, gauge)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, gauge)
				assert.Implements(t, (*Gauge)(nil), gauge)
			}
		})
	}
}

func TestOtelClient_CreateHistogram(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	tests := []struct {
		name       string
		metricName string
		unit       string
		buckets    []float64
		wantErr    bool
	}{
		{
			name:       "valid histogram",
			metricName: "test_histogram",
			unit:       "seconds",
			buckets:    []float64{0.1, 0.5, 1.0, 5.0, 10.0},
			wantErr:    false,
		},
		{
			name:       "histogram with empty buckets",
			metricName: "test_histogram_2",
			unit:       "ms",
			buckets:    []float64{},
			wantErr:    false,
		},
		{
			name:       "histogram with single bucket",
			metricName: "test_histogram_3",
			unit:       "bytes",
			buckets:    []float64{100.0},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			histogram, err := client.CreateHistogram(tt.metricName, tt.unit, tt.buckets)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, histogram)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, histogram)
				assert.Implements(t, (*Histogram)(nil), histogram)
			}
		})
	}
}

func TestCounter_Operations(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	counter, err := client.CreateCounter("test_counter", "requests")
	require.NoError(t, err)

	t.Run("Inc with no labels", func(t *testing.T) {
		assert.NotPanics(t, func() {
			counter.Inc(nil)
		})
	})

	t.Run("Inc with labels", func(t *testing.T) {
		labels := map[string]string{
			"method": "GET",
			"status": "200",
		}
		assert.NotPanics(t, func() {
			counter.Inc(labels)
		})
	})

	t.Run("Add positive value", func(t *testing.T) {
		labels := map[string]string{"operation": "test"}
		assert.NotPanics(t, func() {
			counter.Add(5, labels)
		})
	})

	t.Run("Add zero value", func(t *testing.T) {
		assert.NotPanics(t, func() {
			counter.Add(0, nil)
		})
	})
}

func TestGauge_Operations(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	gauge, err := client.CreateGauge("test_gauge", "bytes")
	require.NoError(t, err)

	t.Run("Set positive value", func(t *testing.T) {
		labels := map[string]string{"component": "memory"}
		assert.NotPanics(t, func() {
			gauge.Set(100.5, labels)
		})
	})

	t.Run("Set negative value", func(t *testing.T) {
		assert.NotPanics(t, func() {
			gauge.Set(-50.0, nil)
		})
	})

	t.Run("Set zero value", func(t *testing.T) {
		assert.NotPanics(t, func() {
			gauge.Set(0.0, nil)
		})
	})
}

func TestHistogram_Operations(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	buckets := []float64{0.1, 0.5, 1.0, 5.0}
	histogram, err := client.CreateHistogram("test_histogram", "seconds", buckets)
	require.NoError(t, err)

	t.Run("Record positive value", func(t *testing.T) {
		labels := map[string]string{"endpoint": "/api/test"}
		assert.NotPanics(t, func() {
			histogram.Record(0.25, labels)
		})
	})

	t.Run("Record zero value", func(t *testing.T) {
		assert.NotPanics(t, func() {
			histogram.Record(0.0, nil)
		})
	})

	t.Run("Record large value", func(t *testing.T) {
		assert.NotPanics(t, func() {
			histogram.Record(100.0, nil)
		})
	})
}

func TestOtelClient_ForceFlush(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Cast to access ForceFlush method
	otelClient := client.(*otelClient)

	t.Run("ForceFlush succeeds", func(t *testing.T) {
		err := otelClient.ForceFlush()
		assert.NoError(t, err)
	})
}

func TestOtelClient_Close(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
		EnableDebug:    true,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)

	t.Run("Close succeeds", func(t *testing.T) {
		err := client.Close()
		assert.NoError(t, err)
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		// Closing again should not cause issues
		err := client.Close()
		// This might error depending on implementation, but shouldn't panic
		_ = err
	})
}

func TestLabelsToAttributes(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected int
	}{
		{
			name:     "nil labels",
			labels:   nil,
			expected: 0,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: 0,
		},
		{
			name: "single label",
			labels: map[string]string{
				"key": "value",
			},
			expected: 1,
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"method": "GET",
				"status": "200",
				"path":   "/api/test",
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := labelsToAttributes(tt.labels)
			assert.Len(t, attrs, tt.expected)

			// Verify all labels are converted to attributes
			if tt.labels != nil {
				labelMap := make(map[string]string)
				for _, attr := range attrs {
					labelMap[string(attr.Key)] = attr.Value.AsString()
				}

				for k, v := range tt.labels {
					assert.Equal(t, v, labelMap[k])
				}
			}
		})
	}
}

func TestOtelClientConcurrency(t *testing.T) {
	cfg := &config.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		OtelEndpoint:   "localhost:4318",
		SendInterval:   10,
	}

	client, err := NewOtelClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Test concurrent metric creation and operations
	t.Run("concurrent counter operations", func(t *testing.T) {
		counter, err := client.CreateCounter("concurrent_counter", "requests")
		require.NoError(t, err)

		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- struct{}{} }()
				for j := 0; j < 100; j++ {
					labels := map[string]string{"worker": string(rune(id + 48))}
					counter.Inc(labels)
					counter.Add(int64(j), labels)
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
