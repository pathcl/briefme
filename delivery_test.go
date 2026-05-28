package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeliverEPUB_CopiesToDestination(t *testing.T) {
	// source EPUB
	src := writeTempEPUBDelivery(t, "fake epub content")

	// simulate a mounted Kobo (just a temp dir)
	koboDir := t.TempDir()

	if err := DeliverEPUB(koboDir, src); err != nil {
		t.Fatalf("DeliverEPUB error: %v", err)
	}

	dst := filepath.Join(koboDir, filepath.Base(src))
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("destination file not found: %v", err)
	}
	if string(data) != "fake epub content" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestDeliverEPUB_MissingKoboPath(t *testing.T) {
	src := writeTempEPUBDelivery(t, "fake epub content")
	err := DeliverEPUB("/nonexistent/kobo/path", src)
	if err == nil {
		t.Fatal("expected error for missing kobo path")
	}
}

func writeTempEPUBDelivery(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "briefme-deliver-*.epub")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}
