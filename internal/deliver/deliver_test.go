package deliver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pathcl/briefme/internal/deliver"
)

func TestToKobo_CopiesToDestination(t *testing.T) {
	src := writeTempEPUB(t, "fake epub content")
	koboDir := t.TempDir()

	if err := deliver.ToKobo(koboDir, src); err != nil {
		t.Fatalf("ToKobo error: %v", err)
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

func TestToKobo_MissingKoboPath(t *testing.T) {
	src := writeTempEPUB(t, "fake epub content")
	if err := deliver.ToKobo("/nonexistent/kobo/path", src); err == nil {
		t.Fatal("expected error for missing kobo path")
	}
}

func writeTempEPUB(t *testing.T, content string) string {
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
