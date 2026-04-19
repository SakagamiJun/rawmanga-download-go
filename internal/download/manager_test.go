package download

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestDownloadPageWithRetryEventuallySucceeds(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		current := attempts.Add(1)
		if current < 3 {
			http.Error(writer, "temporary failure", http.StatusBadGateway)
			return
		}

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("image-bytes"))
	}))
	defer server.Close()

	manager := &Manager{
		client: server.Client(),
	}

	tempDir := t.TempDir()
	finalPath := filepath.Join(tempDir, "page-001.jpg")
	tmpPath := finalPath + ".part"

	size, err := manager.downloadPageWithRetry(server.URL, tmpPath, finalPath, 3)
	if err != nil {
		t.Fatalf("downloadPageWithRetry returned error: %v", err)
	}

	if size != int64(len("image-bytes")) {
		t.Fatalf("unexpected size: %d", size)
	}

	if attempts.Load() != 3 {
		t.Fatalf("unexpected attempts: %d", attempts.Load())
	}

	content, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("read final file: %v", err)
	}

	if string(content) != "image-bytes" {
		t.Fatalf("unexpected file content: %q", string(content))
	}
}

func TestDownloadPageWithRetryExhaustsAttempts(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		http.Error(writer, "temporary failure", http.StatusBadGateway)
	}))
	defer server.Close()

	manager := &Manager{
		client: server.Client(),
	}

	tempDir := t.TempDir()
	finalPath := filepath.Join(tempDir, "page-001.jpg")
	tmpPath := finalPath + ".part"

	if _, err := manager.downloadPageWithRetry(server.URL, tmpPath, finalPath, 2); err == nil {
		t.Fatal("expected retry exhaustion error")
	}

	if attempts.Load() != 3 {
		t.Fatalf("unexpected attempts: %d", attempts.Load())
	}

	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatalf("expected final file to not exist, got err=%v", err)
	}
}
