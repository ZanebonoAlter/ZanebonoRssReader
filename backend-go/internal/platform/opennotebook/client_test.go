package opennotebook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientSummarizeDigest(t *testing.T) {
	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		switch {
		case r.Method == http.MethodGet && r.URL.Path == transformationsPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{
				"id":"transformation:simple",
				"name":"Simple Summary",
				"apply_default":false
			}]`))
		case r.Method == http.MethodPost && r.URL.Path == executeTransformationPath:
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			var req executeTransformationRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Equal(t, "transformation:simple", req.TransformationID)
			require.Equal(t, "gpt-4.1-mini", req.ModelID)
			require.Equal(t, "# markdown", req.InputText)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"output":"# 二次总结","transformation_id":"transformation:simple","model_id":"gpt-4.1-mini"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-key", "gpt-4.1-mini")

	resp, err := client.SummarizeDigest(SummarizeDigestRequest{
		Title:          "今日日报",
		Content:        "# markdown",
		TargetNotebook: "lab-notes",
		PromptMode:     "digest_summary",
	})

	require.NoError(t, err)
	require.Equal(t, "# 二次总结", resp.SummaryMarkdown)
	require.Equal(t, 2, step)
}

func TestClientSummarizeDigest_ReturnsReadableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == transformationsPath {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
			return
		}

		w.WriteHeader(http.StatusBadGateway)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"upstream failed"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-key", "gpt-4.1-mini")

	resp, err := client.SummarizeDigest(SummarizeDigestRequest{
		Title:   "今日日报",
		Content: "# markdown",
	})

	require.Nil(t, resp)
	require.Error(t, err)
	require.ErrorContains(t, err, "no suitable transformation found")
}

func TestNewClientAddsHTTPForBareHost(t *testing.T) {
	client := NewClient("192.168.5.27:5055", "secret-key", "gpt-4.1-mini")

	require.Equal(t, "http://192.168.5.27:5055", client.BaseURL)
}
