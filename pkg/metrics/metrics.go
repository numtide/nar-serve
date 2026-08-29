// Package metrics reports what an instance is spending its bandwidth on, in a
// form Prometheus can scrape.
package metrics

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// InFlight is how many requests are being served right now. Walking an
	// archive is the expensive part of one, so this is what saturation looks
	// like.
	InFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nar_serve_requests_in_flight",
		Help: "Requests currently being served.",
	})

	// Requests counts what clients asked for and what they got.
	Requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nar_serve_requests_total",
		Help: "Requests served, by response status code.",
	}, []string{"code"})

	// UpstreamBytes is what was pulled from the binary cache. Read against
	// ResponseBytes, it says how much of a transfer the client saw the benefit
	// of, which is the cost of serving one file out of an archive.
	UpstreamBytes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nar_serve_upstream_bytes_total",
		Help: "Bytes read from the binary cache.",
	})

	// ArchiveBytes is what came back out of the decompressor, so against
	// UpstreamBytes it gives the compression ratio actually being paid for.
	ArchiveBytes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nar_serve_archive_bytes_total",
		Help: "Bytes of NAR read after decompression.",
	})

	// ResponseBytes is what was written back to clients.
	ResponseBytes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nar_serve_response_bytes_total",
		Help: "Bytes written to clients.",
	})
)

// Count wraps a reader so that whatever is read through it is added to counter.
func Count(r io.Reader, counter prometheus.Counter) io.Reader {
	return &countingReader{reader: r, counter: counter}
}

type countingReader struct {
	reader  io.Reader
	counter prometheus.Counter
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.counter.Add(float64(n))

	return n, err
}

// Middleware records each request as it is served.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		InFlight.Inc()
		defer InFlight.Dec()

		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(wrapped, r)

		status := wrapped.Status()
		if status == 0 {
			// A handler that wrote nothing at all still answered 200.
			status = http.StatusOK
		}

		Requests.WithLabelValues(strconv.Itoa(status)).Inc()
		ResponseBytes.Add(float64(wrapped.BytesWritten()))
	})
}
