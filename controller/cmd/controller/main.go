// Command controller is the read-only gNMI controller from README.md
// section 7 / Phase 3: it subscribes to every device in a registry,
// normalizes updates into a state cache, and emits structured audit
// events and Prometheus metrics. It does not write to devices -- Phase 4
// adds gNMI Set / approval-gated remediation on top of this.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fmolinar/Ohtli/controller/internal/audit"
	"github.com/fmolinar/Ohtli/controller/internal/devices"
	"github.com/fmolinar/Ohtli/controller/internal/gnmi"
	"github.com/fmolinar/Ohtli/controller/internal/metrics"
	"github.com/fmolinar/Ohtli/controller/internal/telemetry"
)

var defaultPaths = []string{
	"/interfaces/interface/state/oper-status",
	"/interfaces/interface/state/counters",
	"/network-instances/network-instance/protocols",
	"/components/component/state",
}

func main() {
	registryPath := flag.String("registry", "devices.yaml", "path to the device registry YAML file")
	listen := flag.String("listen", ":9400", "address to serve /healthz, /cache, and /metrics on")
	reconnectDelay := flag.Duration("reconnect-delay", 5*time.Second, "delay before retrying a failed Subscribe stream")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	auditLog := audit.NewLogger()

	reg, err := devices.Load(*registryPath)
	if err != nil {
		log.Error("failed to load device registry", "error", err, "path", *registryPath)
		os.Exit(1)
	}
	if len(reg.Targets) == 0 {
		log.Warn("device registry has no targets; controller will idle", "path", *registryPath)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cache := telemetry.NewCache()

	go serveHTTP(*listen, cache, log)

	var wg sync.WaitGroup
	for _, target := range reg.Targets {
		wg.Add(1)
		go func(t devices.Target) {
			defer wg.Done()
			runTarget(ctx, t, cache, auditLog, log, *reconnectDelay)
		}(target)
	}

	// Block until shutdown is requested, not just until every target
	// goroutine exits -- with zero (or all failed) targets, wg.Wait()
	// alone would return immediately and tear down the HTTP server too.
	<-ctx.Done()
	log.Info("shutdown requested, waiting for target sessions to close")
	wg.Wait()
}

func runTarget(ctx context.Context, t devices.Target, cache *telemetry.Cache, auditLog *audit.Logger, log *slog.Logger, reconnectDelay time.Duration) {
	tlsCfg := gnmi.TLSConfig{
		Insecure:           t.TLS.Insecure,
		InsecureSkipVerify: t.TLS.InsecureSkipVerify,
		CAFile:             t.TLS.CAFile,
		CertFile:           t.TLS.CertFile,
		KeyFile:            t.TLS.KeyFile,
	}

	for {
		if ctx.Err() != nil {
			return
		}
		if err := connectAndSubscribe(ctx, t, tlsCfg, cache, auditLog, log); err != nil && ctx.Err() == nil {
			log.Warn("target session ended, will retry", "target", t.Name, "error", err)
			metrics.GNMIReconnectsTotal.WithLabelValues(t.Name).Inc()
		}
		metrics.GNMISessionUp.WithLabelValues(t.Name).Set(0)

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func connectAndSubscribe(ctx context.Context, t devices.Target, tlsCfg gnmi.TLSConfig, cache *telemetry.Cache, auditLog *audit.Logger, log *slog.Logger) error {
	client, err := gnmi.Dial(ctx, t.Address, tlsCfg)
	if err != nil {
		auditLog.Emit(ctx, audit.Event{Device: t.Name, EventType: audit.EventError, Operation: "dial", Outcome: audit.OutcomeFailure, NewValue: err.Error()})
		return err
	}
	defer client.Close()

	if err := doCapabilities(ctx, client, t.Name, auditLog); err != nil {
		return err
	}
	if err := doGet(ctx, client, t.Name, t.Paths, auditLog); err != nil {
		return err
	}

	metrics.GNMISessionUp.WithLabelValues(t.Name).Set(1)
	log.Info("subscribed", "target", t.Name, "paths", t.Paths)

	return client.Subscribe(ctx, t.Paths, func(notif *gnmipb.Notification) {
		metrics.SubscriptionUpdatesTotal.WithLabelValues(t.Name).Inc()

		for _, upd := range notif.Update {
			path := gnmi.PathToString(upd.Path)
			old, _ := cache.Get(t.Name, path)
			newVal := gnmi.ValueToInterface(upd.Val)

			cache.Update(t.Name, &gnmipb.Notification{
				Timestamp: notif.Timestamp,
				Prefix:    notif.Prefix,
				Update:    []*gnmipb.Update{upd},
			})

			auditLog.Emit(ctx, audit.Event{
				Device:    t.Name,
				Path:      path,
				EventType: audit.EventUpdate,
				OldValue:  old.Data,
				NewValue:  newVal,
				Operation: "subscribe_update",
				Outcome:   audit.OutcomeSuccess,
			})
		}
		for _, del := range notif.Delete {
			cache.Update(t.Name, &gnmipb.Notification{Timestamp: notif.Timestamp, Prefix: notif.Prefix, Delete: []*gnmipb.Path{del}})
		}
	})
}

func doCapabilities(ctx context.Context, client *gnmi.Client, target string, auditLog *audit.Logger) error {
	start := time.Now()
	_, err := client.Capabilities(ctx)
	duration := time.Since(start)
	metrics.GNMIRequestDuration.WithLabelValues(target, "capabilities").Observe(duration.Seconds())

	outcome := audit.OutcomeSuccess
	var errMsg interface{}
	if err != nil {
		outcome = audit.OutcomeFailure
		errMsg = err.Error()
	}
	auditLog.Emit(ctx, audit.Event{
		Device: target, EventType: audit.EventCapabilities, Operation: "capabilities",
		Outcome: outcome, NewValue: errMsg, Duration: duration,
	})
	return err
}

func doGet(ctx context.Context, client *gnmi.Client, target string, paths []string, auditLog *audit.Logger) error {
	if len(paths) == 0 {
		paths = defaultPaths
	}

	start := time.Now()
	_, err := client.Get(ctx, paths)
	duration := time.Since(start)
	metrics.GNMIRequestDuration.WithLabelValues(target, "get").Observe(duration.Seconds())

	outcome := audit.OutcomeSuccess
	var errMsg interface{}
	if err != nil {
		outcome = audit.OutcomeFailure
		errMsg = err.Error()
	}
	auditLog.Emit(ctx, audit.Event{
		Device: target, EventType: audit.EventGet, Operation: "get",
		Outcome: outcome, NewValue: errMsg, Duration: duration,
	})
	return err
}

func serveHTTP(addr string, cache *telemetry.Cache, log *slog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/cache", func(w http.ResponseWriter, r *http.Request) {
		out := make(map[string]map[string]telemetry.Value)
		for _, dev := range cache.Devices() {
			out[dev] = cache.Snapshot(dev)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			log.Error("failed to encode cache snapshot", "error", err)
		}
	})
	mux.Handle("/metrics", promhttp.Handler())

	log.Info("controller http server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // lab tool, no external timeouts required
		log.Error("http server exited", "error", err)
	}
}
