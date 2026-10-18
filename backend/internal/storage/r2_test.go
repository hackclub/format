package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestR2SDKRoundTrip(t *testing.T) {
	payload := "image bytes"
	exists := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test-bucket/ab/test.jpg" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("request was not signed")
		}
		switch r.Method {
		case http.MethodHead:
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Length", "11")
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != payload {
				t.Errorf("upload body = %q, error = %v", body, err)
			}
			if r.Header.Get("Content-Type") != "image/jpeg" {
				t.Error("missing image content type")
			}
			exists = true
			w.Header().Set("ETag", `"test-etag"`)
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	client, err := NewR2Client(ctx, "test-account", "test-key", "test-secret", "test-bucket", server.URL, "https://cdn.example.test")
	if err != nil {
		t.Fatal(err)
	}
	found, err := client.ObjectExists(ctx, "ab/test.jpg")
	if err != nil || found {
		t.Fatalf("missing object: exists=%v, err=%v", found, err)
	}
	result, err := client.Upload(ctx, "ab/test.jpg", []byte(payload), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if result.URL != "https://cdn.example.test/ab/test.jpg" || result.ETag != `"test-etag"` {
		t.Fatalf("unexpected upload result: %+v", result)
	}
	found, err = client.ObjectExists(ctx, "ab/test.jpg")
	if err != nil || !found {
		t.Fatalf("uploaded object: exists=%v, err=%v", found, err)
	}
}
