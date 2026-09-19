// Package metrics defines the Prometheus metrics README.md section 8
// asks the controller to expose (gNMI session status/reconnects,
// subscription update counts, and gNMI request latency for the Phase 3
// read-only controller; remediation/detection/recovery metrics are
// deferred to Phase 4, once there's a remediation loop to measure).
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	GNMIRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ohtli",
		Subsystem: "controller",
		Name:      "gnmi_request_duration_seconds",
		Help:      "Duration of gNMI RPCs by target and RPC name.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"target", "rpc"})

	GNMISessionUp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ohtli",
		Subsystem: "controller",
		Name:      "gnmi_session_up",
		Help:      "1 if the gNMI session to a target is currently established, 0 otherwise.",
	}, []string{"target"})

	GNMIReconnectsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ohtli",
		Subsystem: "controller",
		Name:      "gnmi_reconnects_total",
		Help:      "Number of times the gNMI Subscribe stream to a target had to reconnect.",
	}, []string{"target"})

	SubscriptionUpdatesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ohtli",
		Subsystem: "controller",
		Name:      "subscription_updates_total",
		Help:      "Number of gNMI Subscribe updates received, by target.",
	}, []string{"target"})
)
