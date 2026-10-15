package clients

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	httptransport "github.com/go-openapi/runtime/client"
)

type captureLogger struct {
	msgs []string
}

func (c *captureLogger) Info(msg string, keysAndValues ...interface{}) {
	c.msgs = append(c.msgs, msg+": "+formatKV(keysAndValues...))
}
func (c *captureLogger) Debug(msg string, keysAndValues ...interface{}) {
	c.msgs = append(c.msgs, "debug "+msg+": "+formatKV(keysAndValues...))
}
func (c *captureLogger) Error(msg string, keysAndValues ...interface{}) {
	c.msgs = append(c.msgs, "error "+msg+": "+formatKV(keysAndValues...))
}
func (c *captureLogger) WithValues(keysAndValues ...interface{}) logging.Logger {
	return c
}
func (c *captureLogger) WithName(name string) logging.Logger {
	return c
}

func formatKV(kvs ...interface{}) string {
	var b strings.Builder
	for i := 0; i+1 < len(kvs); i += 2 {
		_, _ = fmt.Fprintf(&b, " %v=%v", kvs[i], kvs[i+1])
	}
	return b.String()
}

func TestLoggingRoundTripperCapturesErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"code":"NOT_FOUND","message":"project not found"}]}`))
		_ = body
	}))
	defer srv.Close()

	cl := &captureLogger{}
	rt := &loggingRoundTripper{base: http.DefaultTransport, logger: cl}
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v2.0/projects/3/members", bytes.NewReader([]byte(`{"role_id":2}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	got, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(got), "project not found") {
		t.Errorf("response body not preserved: %s", got)
	}
	if len(cl.msgs) == 0 {
		t.Fatal("expected error log")
	}
	joined := strings.Join(cl.msgs, "\n")
	if !strings.Contains(joined, "project not found") {
		t.Errorf("log missing response body: %s", joined)
	}
	if !strings.Contains(joined, "status=404") {
		t.Errorf("log missing status: %s", joined)
	}
}

func TestNewHarborClientWrapsTransport(t *testing.T) {
	c, err := NewHarborClient(&HarborConfig{URL: "http://127.0.0.1:1", Username: "a", Password: "b"})
	if err != nil {
		t.Fatal(err)
	}
	api := c.clientSet.V2()
	if api == nil {
		t.Fatal("nil v2")
	}
	rt, ok := api.Transport.(*httptransport.Runtime)
	if !ok {
		t.Fatalf("Transport is %T, want *httptransport.Runtime", api.Transport)
	}
	if _, ok := rt.Transport.(*loggingRoundTripper); !ok {
		t.Fatalf("rt.Transport is %T, want *loggingRoundTripper", rt.Transport)
	}
}
