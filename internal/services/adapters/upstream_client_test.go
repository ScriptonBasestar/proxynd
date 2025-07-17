package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/services/proxy"
)

func TestNewHTTPUpstreamClient(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "zero timeout",
			timeout: 0,
		},
		{
			name:    "custom timeout",
			timeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewHTTPUpstreamClient(tt.timeout)
			assert.NotNil(t, client)
			assert.NotNil(t, client.client)

			assert.Equal(t, tt.timeout, client.client.Timeout)
		})
	}
}

func TestHTTPUpstreamClient_Fetch(t *testing.T) {
	tests := []struct {
		name           string
		setupServer    func() *httptest.Server
		headers        map[string]string
		wantStatusCode int
		wantErr        bool
		errMsg         string
		checkResponse  func(*testing.T, *proxy.ProxyResponse)
	}{
		{
			name: "successful fetch",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.Header().Set("X-Custom-Header", "custom-value")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("test content"))
				}))
			},
			headers: map[string]string{
				"User-Agent": "test-agent",
			},
			wantStatusCode: 200,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *proxy.ProxyResponse) {
				assert.Equal(t, "text/plain", resp.ContentType)
				assert.Equal(t, "custom-value", resp.Headers["X-Custom-Header"])

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(body))
			},
		},
		{
			name: "404 response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte("not found"))
				}))
			},
			wantStatusCode: 404,
			wantErr:        true,
			errMsg:         "upstream returned error: 404",
		},
		{
			name: "with custom headers",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Echo back the received headers
					assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
					assert.Equal(t, "custom-agent", r.Header.Get("User-Agent"))
					w.WriteHeader(http.StatusOK)
				}))
			},
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "custom-agent",
			},
			wantStatusCode: 200,
			wantErr:        false,
		},
		{
			name: "server timeout",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(100 * time.Millisecond)
					w.WriteHeader(http.StatusOK)
				}))
			},
			wantErr: true,
			errMsg:  "context deadline exceeded",
		},
		{
			name: "invalid URL",
			setupServer: func() *httptest.Server {
				return nil
			},
			wantErr: true,
			errMsg:  "missing protocol scheme",
		},
		{
			name: "large response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Send 1MB of data
					data := make([]byte, 1024*1024)
					for i := range data {
						data[i] = byte(i % 256)
					}
					w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
					w.WriteHeader(http.StatusOK)
					w.Write(data)
				}))
			},
			wantStatusCode: 200,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *proxy.ProxyResponse) {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, 1024*1024, len(body))
			},
		},
		{
			name: "with content disposition",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Disposition", `attachment; filename="test.jar"`)
					w.WriteHeader(http.StatusOK)
				}))
			},
			wantStatusCode: 200,
			wantErr:        false,
			checkResponse: func(t *testing.T, resp *proxy.ProxyResponse) {
				assert.Equal(t, "test.jar", resp.FileName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			if tt.setupServer != nil {
				server := tt.setupServer()
				if server != nil {
					defer server.Close()
					url = server.URL + "/test/path"
				} else {
					url = "://invalid-url"
				}
			}

			// Create client with short timeout for timeout test
			timeout := 30 * time.Second
			if tt.name == "server timeout" {
				timeout = 50 * time.Millisecond
			}
			client := NewHTTPUpstreamClient(timeout)

			ctx := context.Background()
			if tt.name == "server timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			resp, err := client.Fetch(ctx, url, tt.headers)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				// For upstream errors, we still return a response with error details
				if strings.Contains(err.Error(), "upstream returned error") {
					require.NotNil(t, resp)
					assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
				} else {
					assert.Nil(t, resp)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
				assert.False(t, resp.Cached)

				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestHTTPUpstreamClient_FetchWithRedirect(t *testing.T) {
	redirectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if redirectCount < 2 {
			redirectCount++
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final content"))
	}))
	defer server.Close()

	client := NewHTTPUpstreamClient(0)
	ctx := context.Background()
	resp, err := client.Fetch(ctx, server.URL+"/start", nil)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "final content", string(body))
}

func TestHTTPUpstreamClient_FetchWithContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPUpstreamClient(0)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	resp, err := client.Fetch(ctx, server.URL, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, resp)
}

func TestHTTPUpstreamClient_ParseContentDisposition(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		wantFile string
	}{
		{
			name:     "simple filename",
			header:   `attachment; filename="test.jar"`,
			wantFile: "test.jar",
		},
		{
			name:     "filename without quotes",
			header:   `attachment; filename=test.jar`,
			wantFile: "test.jar",
		},
		{
			name:     "filename with spaces",
			header:   `attachment; filename="my file.jar"`,
			wantFile: "my file.jar",
		},
		{
			name:     "inline disposition",
			header:   `inline; filename="doc.pdf"`,
			wantFile: "doc.pdf",
		},
		{
			name:     "no filename",
			header:   `attachment`,
			wantFile: "",
		},
		{
			name:     "empty header",
			header:   "",
			wantFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.header != "" {
					w.Header().Set("Content-Disposition", tt.header)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewHTTPUpstreamClient(0)
			ctx := context.Background()
			resp, err := client.Fetch(ctx, server.URL, nil)

			assert.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantFile, resp.FileName)
		})
	}
}

// Benchmark test
func BenchmarkHTTPUpstreamClient_Fetch(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark content"))
	}))
	defer server.Close()

	client := NewHTTPUpstreamClient(0)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Fetch(ctx, server.URL, nil)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
