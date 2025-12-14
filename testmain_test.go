package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var webhookBinPath string
var webhookTempDir string

// TestMain builds the webhook binary once for expensive integration tests.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "webhook-test-bin-")
	if err != nil {
		panic(err)
	}
	webhookTempDir = tmp
	bin := filepath.Join(tmp, "webhook")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin)
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmp)
		panic(err)
	}
	webhookBinPath = bin

	code := m.Run()

	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

func webhookBinary(t *testing.T) string {
	if webhookBinPath == "" {
		t.Fatal("webhook binary not built")
	}
	return webhookBinPath
}

