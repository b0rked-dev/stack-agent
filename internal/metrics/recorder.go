package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Recorder is the interface for recording agent metrics.
type Recorder interface {
	RecordPoll(stack, result string)
	RecordDeploy(stack, result string, duration time.Duration)
}

// NoopRecorder is a Recorder that does nothing. Useful in tests.
type NoopRecorder struct{}

// RecordPoll is a no-op implementation.
func (n *NoopRecorder) RecordPoll(_, _ string) {}

// RecordDeploy is a no-op implementation.
func (n *NoopRecorder) RecordDeploy(_, _ string, _ time.Duration) {}

// PrometheusRecorder records metrics into a private prometheus registry.
type PrometheusRecorder struct {
	reg             *prometheus.Registry
	pollsTotal      *prometheus.CounterVec
	deploysTotal    *prometheus.CounterVec
	deployDuration  *prometheus.HistogramVec
	lastDeployStamp *prometheus.GaugeVec
}

// NewPrometheusRecorder creates a PrometheusRecorder backed by a private registry.
func NewPrometheusRecorder() *PrometheusRecorder {
	reg := prometheus.NewRegistry()

	pollsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stackagent_polls_total",
		Help: "Total number of poll iterations, partitioned by stack and result.",
	}, []string{"stack", "result"})

	deploysTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stackagent_deploys_total",
		Help: "Total number of deploys attempted, partitioned by stack and result.",
	}, []string{"stack", "result"})

	deployDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "stackagent_deploy_duration_seconds",
		Help:    "Duration of deploy operations in seconds.",
		Buckets: []float64{1, 5, 10, 30, 60, 120},
	}, []string{"stack"})

	lastDeployStamp := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stackagent_last_deploy_timestamp_seconds",
		Help: "Unix timestamp of the last deploy for each stack.",
	}, []string{"stack"})

	reg.MustRegister(pollsTotal, deploysTotal, deployDuration, lastDeployStamp)

	return &PrometheusRecorder{
		reg:             reg,
		pollsTotal:      pollsTotal,
		deploysTotal:    deploysTotal,
		deployDuration:  deployDuration,
		lastDeployStamp: lastDeployStamp,
	}
}

// RecordPoll records a poll iteration result for the given stack.
func (r *PrometheusRecorder) RecordPoll(stack, result string) {
	r.pollsTotal.WithLabelValues(stack, result).Inc()
}

// RecordDeploy records a deploy event for the given stack with duration.
func (r *PrometheusRecorder) RecordDeploy(stack, result string, duration time.Duration) {
	r.deploysTotal.WithLabelValues(stack, result).Inc()
	r.deployDuration.WithLabelValues(stack).Observe(duration.Seconds())
	r.lastDeployStamp.WithLabelValues(stack).SetToCurrentTime()
}

// Registry returns the private prometheus registry used by this recorder.
func (r *PrometheusRecorder) Registry() *prometheus.Registry {
	return r.reg
}
