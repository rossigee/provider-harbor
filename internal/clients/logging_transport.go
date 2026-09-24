/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clients

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

// maxLoggedBody caps request/response bodies written to logs (bytes).
const maxLoggedBody = 4096

// loggingRoundTripper wraps an http.RoundTripper and logs non-2xx
// Harbor API responses with their response bodies. go-swagger's
// default-case APIError marshals the unexported client.response
// struct to "{}", so the real Harbor error body is otherwise lost
// after Submit closes the body.
type loggingRoundTripper struct {
	base   http.RoundTripper
	logger logging.Logger
}

func (t *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	var reqBody []byte
	if req.Body != nil && req.Method != http.MethodGet && req.Method != http.MethodHead {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		req.ContentLength = int64(len(reqBody))
		if req.GetBody == nil {
			body := reqBody
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(body)), nil
			}
		}
	}

	res, err := base.RoundTrip(req)
	if err != nil {
		if t.logger != nil {
			t.logger.Info("Harbor API request failed",
				"method", req.Method,
				"path", sanitizeForLog(req.URL.RequestURI()),
				"error", err.Error(),
			)
		}
		return nil, err
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}

	respBody, readErr := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if readErr != nil {
		if t.logger != nil {
			t.logger.Info("Harbor API error response (body unreadable)",
				"method", req.Method,
				"path", sanitizeForLog(req.URL.RequestURI()),
				"status", res.StatusCode,
				"error", readErr.Error(),
			)
		}
		res.Body = io.NopCloser(bytes.NewReader(nil))
		return res, nil
	}
	res.Body = io.NopCloser(bytes.NewReader(respBody))
	res.ContentLength = int64(len(respBody))

	if t.logger != nil {
		t.logger.Info("Harbor API error response",
			"method", req.Method,
			"path", sanitizeForLog(req.URL.RequestURI()),
			"status", res.StatusCode,
			"requestBody", truncateForLog(reqBody),
			"responseBody", truncateForLog(respBody),
		)
	}
	return res, nil
}

func truncateForLog(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) > maxLoggedBody {
		return sanitizeForLog(string(b[:maxLoggedBody])) + "...(truncated)"
	}
	return sanitizeForLog(string(b))
}

func sanitizeForLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
