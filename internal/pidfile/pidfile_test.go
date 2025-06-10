package pidfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPIDFile(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "pidfile_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up
	tmpfile.Close()                 // close it so New can write to it

	// Test New
	pidFile, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Test Write
	if err := pidFile.Write(); err != nil {
		t.Fatal(err)
	}

	// Test Remove
	if err := pidFile.Remove(); err != nil {
		t.Fatal(err)
	}
}

func TestNewAndRemove(t *testing.T) {
	// Create a temporary dir
	tmpDir, err := os.MkdirTemp("", "pidfile-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pidFilePath := filepath.Join(tmpDir, "test.pid")

	// Test New
	pidFile, err := New(pidFilePath)
	if err != nil {
		t.Fatal("Could not create test file", err)
	}

	_, err = New(pidFilePath)
	if err == nil {
		t.Fatal("Test file creation not blocked")
	}

	if err := pidFile.Remove(); err != nil {
		t.Fatal("Could not delete created test file")
	}
}

func TestRemoveInvalidPath(t *testing.T) {
	file := PIDFile{path: filepath.Join("foo", "bar")}

	if err := file.Remove(); err == nil {
		t.Fatal("Non-existing file doesn't give an error on delete")
	}
}
