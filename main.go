package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const version = "1.0.0"

var (
	startTime = time.Now()
	registry  = prometheus.NewRegistry()

	appRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "app",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests handled by the demo app.",
	}, []string{"method", "path", "status"})

	appRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "app",
		Name:      "request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	appRequestsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "app",
		Name:      "requests_in_flight",
		Help:      "Number of currently in-flight HTTP requests.",
	})

	appUptimeSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "app",
		Name:      "uptime_seconds",
		Help:      "Application uptime in seconds.",
	})

	appGoRoutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "app",
		Name:      "go_routines",
		Help:      "Number of active goroutines in the demo app.",
	})

	appMemoryAllocBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "app",
		Name:      "memory_alloc_bytes",
		Help:      "Number of bytes allocated by the demo app.",
	})
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func init() {
	registry.MustRegister(
		appRequestsTotal,
		appRequestDuration,
		appRequestsInFlight,
		appUptimeSeconds,
		appGoRoutines,
		appMemoryAllocBytes,
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
}

func recordRuntimeMetrics() {
	var memStats runtime.MemStats

	for range time.NewTicker(10 * time.Second).C {
		runtime.ReadMemStats(&memStats)
		appMemoryAllocBytes.Set(float64(memStats.Alloc))
		appGoRoutines.Set(float64(runtime.NumGoroutine()))
		appUptimeSeconds.Set(time.Since(startTime).Seconds())
	}
}

func instrumentHandler(path string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appRequestsInFlight.Inc()
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			appRequestsInFlight.Dec()
			appRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
			appRequestsTotal.WithLabelValues(r.Method, path, fmt.Sprintf("%d", rw.status)).Inc()
		}()
		handler.ServeHTTP(rw, r)
	})
}

func renderIndex(personName, hostname string) string {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	personName = html.EscapeString(personName)
	hostname = html.EscapeString(hostname)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hello %[1]s from Kubernetes</title>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Red+Hat+Display:wght@700;800&family=Red+Hat+Text:wght@400;700&family=Red+Hat+Mono:wght@400;500&display=swap">
    <style>
        :root {
            --red: #ee0000; --teal: #7cdad3; --lavender: #b6a6e9;
            --text: #f2f2f6; --muted: #b8b6c9; --green: #95d58e;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 24px;
            font-family: 'Red Hat Text', 'Helvetica Neue', Arial, sans-serif;
            color: var(--text);
            background:
                radial-gradient(1000px 700px at -6%% -10%%, rgba(42,27,107,.85), transparent 62%%),
                radial-gradient(800px 520px at 108%% 112%%, rgba(27,17,72,.7), transparent 64%%),
                #000;
            background-attachment: fixed;
        }
        .container {
            position: relative;
            width: 100%%;
            max-width: 760px;
            padding: 40px;
            border: 1.5px solid rgba(124,218,211,.45);
            border-radius: 26px;
            background: linear-gradient(150deg, rgba(94,64,190,.16), rgba(42,27,107,.1));
            box-shadow: 0 24px 60px -44px #5e40be;
        }
        .container::before {
            content: "";
            position: absolute;
            top: 20px; right: 20px;
            width: 11px; height: 11px;
            border-radius: 3px;
            background: var(--red);
            box-shadow: 0 0 16px rgba(238,0,0,.7);
        }
        h1 {
            font-family: 'Red Hat Display', 'Helvetica Neue', Arial, sans-serif;
            font-size: clamp(1.8rem, 5vw, 2.6rem);
            font-weight: 800;
            letter-spacing: -.03em;
            line-height: 1.15;
        }
        h1::after {
            content: "";
            display: block;
            width: 72px; height: 5px;
            margin: 16px 0 28px;
            border-radius: 999px;
            background: linear-gradient(90deg, var(--red), #5e40be);
        }
        .info { display: grid; gap: 12px; }
        .metric {
            display: flex;
            flex-wrap: wrap;
            justify-content: space-between;
            gap: 4px 20px;
            padding: 14px 18px;
            border: 1px solid rgba(182,166,233,.25);
            border-left: 4px solid var(--red);
            border-radius: 14px;
            background: rgba(0,0,0,.45);
        }
        .metric strong {
            font-size: .78rem;
            letter-spacing: .12em;
            text-transform: uppercase;
            color: var(--teal);
        }
        .metric span { font-family: 'Red Hat Mono', ui-monospace, Menlo, monospace; word-break: break-all; }
        .metric .ok { color: var(--green); }
        .footer {
            margin-top: 28px;
            padding-top: 18px;
            border-top: 1px solid rgba(182,166,233,.2);
            font-size: .9rem;
            color: var(--muted);
        }
        code {
            padding: 2px 6px;
            border-radius: 5px;
            background: rgba(255,255,255,.08);
            color: var(--lavender);
            font-family: 'Red Hat Mono', ui-monospace, Menlo, monospace;
        }
        @media (max-width: 520px) { .container { padding: 28px 20px; } }
    </style>
</head>
<body>
    <div class="container">
        <h1>👋 Hello %[1]s from Kubernetes</h1>
        <div class="info">
            <div class="metric"><strong>Pod Hostname</strong> <span>%[2]s</span></div>
            <div class="metric"><strong>Application Version</strong> <span>%[3]s</span></div>
            <div class="metric"><strong>Deployment Time</strong> <span>%[4]s</span></div>
            <div class="metric"><strong>Uptime</strong> <span>%[5]s</span></div>
            <div class="metric"><strong>Goroutines</strong> <span>%[6]d</span></div>
            <div class="metric"><strong>Memory Allocated</strong> <span>%[7]s MB</span></div>
            <div class="metric"><strong>Service Status</strong> <span class="ok">Running ✓</span></div>
        </div>
        <div class="footer">
            Metrics are emitted in real time at <code>/metrics</code> for Prometheus and Grafana.
        </div>
    </div>
</body>
</html>`, personName, hostname, version, time.Now().Format("2006-01-02 15:04:05 MST"), time.Since(startTime).Truncate(time.Second), runtime.NumGoroutine(), fmt.Sprintf("%.1f", float64(memStats.Alloc)/1024.0/1024.0))
}

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	personName := os.Getenv("PERSON_NAME")
	if personName == "" {
		personName = "World"
	}

	go recordRuntimeMetrics()

	http.Handle("/", instrumentHandler("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		hostname, _ := os.Hostname()
		if _, err := fmt.Fprintf(w, "%s", renderIndex(personName, hostname)); err != nil {
			log.Printf("failed to write index response: %v", err)
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})))

	http.Handle("/health", instrumentHandler("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprintf(w, `{"status":"healthy","version":"%s"}`, version); err != nil {
			log.Printf("failed to write health response: %v", err)
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})))

	http.Handle("/pod", instrumentHandler("/pod", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hostname, _ := os.Hostname()
		if _, err := fmt.Fprintf(w, `{"pod_name": "%s"}`+"\n", hostname); err != nil {
			log.Printf("failed to write pod response: %v", err)
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})))

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s with PERSON_NAME=%s\n", port, personName)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
