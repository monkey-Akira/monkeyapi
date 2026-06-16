package common

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbedFileSystemExistsStripsMountedPrefix(t *testing.T) {
	dir := t.TempDir()
	assetDir := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}

	efs := &embedFileSystem{FileSystem: http.Dir(dir)}

	if !efs.Exists("/legacy", "/legacy/assets/app.js") {
		t.Fatal("expected prefixed legacy asset path to resolve")
	}
	if !efs.Exists("/", "/assets/app.js") {
		t.Fatal("expected root-mounted asset path to resolve")
	}
	if efs.Exists("/legacy", "/legacy/assets/missing.js") {
		t.Fatal("expected missing legacy asset path not to resolve")
	}
}
