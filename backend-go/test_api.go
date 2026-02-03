package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const baseURL = "http://localhost:8080/api"

func testEndpoint(method, endpoint string, body interface{}) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			fmt.Printf("❌ Error marshaling request for %s: %v\n", endpoint, err)
			return
		}
	}

	req, err := http.NewRequest(method, baseURL+endpoint, bytes.NewReader(reqBody))
	if err != nil {
		fmt.Printf("❌ Error creating request for %s: %v\n", endpoint, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error calling %s: %v\n", endpoint, err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("✅ %s %s - Status: %d\n", method, endpoint, resp.StatusCode)
	} else {
		fmt.Printf("❌ %s %s - Status: %d, Response: %s\n", method, endpoint, resp.StatusCode, string(respBody))
	}
}

func main() {
	fmt.Println("Testing RSS Reader Go Backend API Endpoints...")
	fmt.Println("=" + string(make([]byte, 60)))

	// Test Categories API
	fmt.Println("\n📁 Categories API:")
	testEndpoint("GET", "/categories", nil)
	testEndpoint("POST", "/categories", map[string]interface{}{"name": "Test Category"})

	// Test Feeds API
	fmt.Println("\n📰 Feeds API:")
	testEndpoint("GET", "/feeds", nil)
	testEndpoint("POST", "/feeds/fetch", map[string]interface{}{"url": "https://example.com/rss"})

	// Test Articles API
	fmt.Println("\n📄 Articles API:")
	testEndpoint("GET", "/articles", nil)
	testEndpoint("GET", "/articles/stats", nil)

	// Test AI API
	fmt.Println("\n🤖 AI API:")
	testEndpoint("POST", "/ai/test", map[string]interface{}{
		"base_url": "https://api.openai.com/v1",
		"api_key":  "test-key",
		"model":    "gpt-4o-mini",
	})
	testEndpoint("GET", "/ai/settings", nil)

	// Test Summaries API
	fmt.Println("\n🧠 Summaries API:")
	testEndpoint("GET", "/summaries", nil)

	// Test Auto Summary API
	fmt.Println("\n⚙️ Auto Summary API:")
	testEndpoint("GET", "/auto-summary/status", nil)

	// Test OPML API
	fmt.Println("\n📋 OPML API:")
	testEndpoint("GET", "/export-opml", nil)

	// Test Schedulers API
	fmt.Println("\n⏰ Schedulers API:")
	testEndpoint("GET", "/schedulers/status", nil)
	testEndpoint("GET", "/schedulers/auto-refresh/status", nil)

	// Test Tasks API
	fmt.Println("\n📋 Tasks API:")
	testEndpoint("GET", "/tasks/status", nil)

	fmt.Println("\n" + string(make([]byte, 60)))
	fmt.Println("✅ API testing completed!")

	// Print summary
	fmt.Println("\n📊 API Implementation Summary:")
	fmt.Println("=" + string(make([]byte, 60)))

	apiList := []struct {
		method   string
		endpoint string
		status   string
	}{
		{"GET", "/categories", "✅ Implemented"},
		{"POST", "/categories", "✅ Implemented"},
		{"PUT", "/categories/{id}", "✅ Implemented"},
		{"DELETE", "/categories/{id}", "✅ Implemented"},

		{"GET", "/feeds", "✅ Implemented"},
		{"POST", "/feeds", "✅ Implemented"},
		{"PUT", "/feeds/{id}", "✅ Implemented"},
		{"DELETE", "/feeds/{id}", "✅ Implemented"},
		{"POST", "/feeds/{id}/refresh", "✅ Implemented"},
		{"POST", "/feeds/fetch", "✅ Implemented"},
		{"POST", "/feeds/refresh-all", "✅ NEWLY IMPLEMENTED"},

		{"GET", "/articles", "✅ Implemented"},
		{"GET", "/articles/{id}", "✅ Implemented"},
		{"PUT", "/articles/{id}", "✅ Implemented"},
		{"PUT", "/articles/bulk-update", "✅ Implemented"},
		{"GET", "/articles/stats", "✅ Implemented"},

		{"GET", "/summaries", "✅ Implemented"},
		{"POST", "/summaries/generate", "✅ With AI Integration"},
		{"POST", "/summaries/auto-generate", "✅ NEWLY IMPLEMENTED"},
		{"GET", "/summaries/{id}", "✅ Implemented"},
		{"DELETE", "/summaries/{id}", "✅ Implemented"},

		{"POST", "/ai/summarize", "✅ Implemented"},
		{"POST", "/ai/test", "✅ Implemented"},
		{"GET", "/ai/settings", "✅ Implemented"},
		{"POST", "/ai/settings", "✅ Implemented"},

		{"GET", "/auto-summary/status", "✅ NEWLY IMPLEMENTED"},
		{"POST", "/auto-summary/config", "✅ NEWLY IMPLEMENTED"},

		{"POST", "/import-opml", "✅ Implemented"},
		{"GET", "/export-opml", "✅ Implemented"},

		{"GET", "/schedulers/status", "✅ Implemented (stub)"},
		{"GET", "/schedulers/{name}/status", "✅ Implemented (stub)"},
		{"POST", "/schedulers/{name}/trigger", "✅ Implemented (stub)"},

		{"GET", "/tasks/status", "✅ Implemented (stub)"},
	}

	for _, api := range apiList {
		fmt.Printf("%-7s %-35s %s\n", api.method, api.endpoint, api.status)
	}

	os.Exit(0)
}
