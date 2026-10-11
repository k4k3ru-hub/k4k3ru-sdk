package httpclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type responseBodyFixture struct {
	io.Reader
	closed bool
	err    error
}

// Close records body cleanup and can simulate an I/O failure.
//
// Version:
//   - 2026-10-11: Added.
func (b *responseBodyFixture) Close() error { b.closed = true; return b.err }

// TestHTTPResponseValidation rejects oversized and malformed envelopes and always closes their bodies.
//
// Version:
//   - 2026-10-11: Added.
func TestHTTPResponseValidation(t *testing.T) {
	for name, raw := range map[string]string{
		"malformed":      `{`,
		"trailing_data":  `{"id":1,"result":{}} {}`,
		"wrong_id":       `{"id":2,"result":{}}`,
		"missing_id":     `{"result":{}}`,
		"missing_result": `{"id":1}`,
		"null_result":    `{"id":1,"result":null}`,
		"oversized":      strings.Repeat(" ", maxResponseBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			body := &responseBodyFixture{Reader: strings.NewReader(raw)}
			client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
				decodeTestRequest(t, r)
				return &http.Response{StatusCode: 200, Body: body}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.MarketHub().Candles().Get(t.Context(), testParams()); err == nil || !body.closed {
				t.Fatalf("malformed response accepted or body leaked: %v", err)
			}
		})
	}
}

// TestHTTPResponseCloseError preserves cleanup failures alongside the original application error.
//
// Version:
//   - 2026-10-11: Added.
func TestHTTPResponseCloseError(t *testing.T) {
	want := errors.New("failed to close test response")
	body := &responseBodyFixture{Reader: strings.NewReader(`{"id":1,"error":{"code":"out_of_ticks"}}`), err: want}
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		decodeTestRequest(t, r)
		return &http.Response{StatusCode: 200, Body: body}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.MarketHub().Candles().Get(t.Context(), testParams()); !errors.Is(err, want) || !strings.Contains(err.Error(), "out_of_ticks") || !body.closed {
		t.Fatalf("cleanup error lost: %v", err)
	}
}
