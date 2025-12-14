package nodeclient

import (
	"fmt"
	"os"
	"path/filepath"
)

func ensureTargetWritable(targetPath string) error {
	clean := filepath.Clean(targetPath)
	if clean == "" || clean == "." || clean == "/" {
		return fmt.Errorf("invalid targetPath: %q", targetPath)
	}

	if err := os.MkdirAll(clean, 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	f, err := os.CreateTemp(clean, ".gohook-write-test-*")
	if err != nil {
		return fmt.Errorf("write test in target dir: %w", err)
	}
	name := f.Name()
	_, werr := f.WriteString("ok")
	cerr := f.Close()
	_ = os.Remove(name)
	if werr != nil {
		return fmt.Errorf("write test failed: %w", werr)
	}
	if cerr != nil {
		return fmt.Errorf("close test file failed: %w", cerr)
	}
	return nil
}
