/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package prometheus

// Deprecated 使用opentelemetry
import (
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	prometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"
)

/*func init() {
	sink, _ := prometheus.NewPrometheusSink()
	conf := metrics.DefaultConfig("")
	metrics1, _ := metrics.New(conf, sink)
	metrics1.EnableHostnameLabel = true
	http.Handle("/metrics", promhttp.Handler())
	reg.MustRegister(srvMetrics)
}*/

var reg = prometheus.NewRegistry()

type MetricsRecord = func(reqTime time.Time, uri, method string, code int)

var defaultMetricsRecord = func(reqTime time.Time, uri, method string, code int) {
	labels := prometheus.Labels{
		"method": method,
		"uri":    uri,
	}
	t := time.Now().Sub(reqTime)
	AccessCounter.With(labels).Add(1)
	QueueGauge.With(labels).Set(1)
	HttpDurationsHistogram.With(labels).Observe(float64(t) / 1000)
	HttpDurations.With(labels).Observe(float64(t) / 1000)
}

func SetMetricsRecord(metricsRecord MetricsRecord) {
	if metricsRecord != nil {
		defaultMetricsRecord = metricsRecord
	}
}

var srvMetrics = grpcprom.NewServerMetrics(
	grpcprom.WithServerHandlingTimeHistogram(
		grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
	),
)

var AccessCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "api_requests_total",
	},
	[]string{"method", "uri"},
)

var QueueGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "queue_num_total",
	},
	[]string{"method", "uri"},
)

var HttpDurationsHistogram = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_durations_histogram_millisecond",
		Buckets: []float64{30, 60, 100, 200, 300, 500, 1000},
	},
	[]string{"method", "uri"},
)
var HttpDurations = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Name:       "http_durations_millisecond",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
	[]string{"method", "uri"},
)

func init() {
	http.Handle("/metrics", promhttp.Handler())
	prometheus.MustRegister(AccessCounter, QueueGauge, HttpDurationsHistogram, HttpDurations)
}
