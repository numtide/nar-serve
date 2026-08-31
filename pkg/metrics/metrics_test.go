package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCount(t *testing.T) {
	before := testutil.ToFloat64(UpstreamBytes)

	n, err := io.Copy(io.Discard, Count(strings.NewReader("0123456789"), UpstreamBytes))
	require.NoError(t, err)
	assert.EqualValues(t, 10, n)

	assert.EqualValues(t, 10, testutil.ToFloat64(UpstreamBytes)-before)
}

// A reader stopped part-way through has still cost what it read, so that is
// what the counter has to show.
func TestCountPartialRead(t *testing.T) {
	before := testutil.ToFloat64(UpstreamBytes)

	buf := make([]byte, 4)
	_, err := io.ReadFull(Count(strings.NewReader("0123456789"), UpstreamBytes), buf)
	require.NoError(t, err)

	assert.EqualValues(t, 4, testutil.ToFloat64(UpstreamBytes)-before)
}

func TestMiddleware(t *testing.T) {
	beforeBytes := testutil.ToFloat64(ResponseBytes)
	before404 := testutil.ToFloat64(Requests.WithLabelValues("404"))

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In flight counts the request being served, so it is up while the
		// handler runs and back down once it returns.
		assert.EqualValues(t, 1, testutil.ToFloat64(InFlight))

		http.Error(w, "file not found", http.StatusNotFound)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/nix/store/whatever", nil))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.EqualValues(t, 0, testutil.ToFloat64(InFlight))
	assert.EqualValues(t, 1, testutil.ToFloat64(Requests.WithLabelValues("404"))-before404)
	assert.EqualValues(t, len("file not found\n"), testutil.ToFloat64(ResponseBytes)-beforeBytes)
}

// A handler that answers without writing anything still answered 200.
func TestMiddlewareEmptyResponse(t *testing.T) {
	before := testutil.ToFloat64(Requests.WithLabelValues("200"))

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	assert.EqualValues(t, 1, testutil.ToFloat64(Requests.WithLabelValues("200"))-before)
}
