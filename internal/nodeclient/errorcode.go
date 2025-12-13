package nodeclient

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
)

type codedError struct {
	Code    string
	Message string
}

func (e codedError) Error() string { return e.Message }

func classifyError(err error) codedError {
	if err == nil {
		return codedError{Code: "", Message: ""}
	}

	code := "UNKNOWN"
	switch {
	case errors.Is(err, syscall.EACCES):
		code = "EACCES"
	case errors.Is(err, syscall.EPERM):
		code = "EPERM"
	case errors.Is(err, syscall.EROFS):
		code = "EROFS"
	case errors.Is(err, syscall.ENOSPC):
		code = "ENOSPC"
	case errors.Is(err, syscall.ENOENT):
		code = "ENOENT"
	}

	// Helpful formatting for common filesystem errors.
	if pe := (*os.PathError)(nil); errors.As(err, &pe) {
		msg := pe.Error()
		switch code {
		case "EACCES", "EPERM", "EROFS":
			msg = fmt.Sprintf("%s (no write permission). Fix: check owner/permissions or choose another targetPath.", msg)
		case "ENOENT":
			msg = fmt.Sprintf("%s (path missing). Fix: ensure parent directory exists or let agent create it.", msg)
		case "ENOSPC":
			msg = fmt.Sprintf("%s (disk full). Fix: free space on target filesystem.", msg)
		}
		return codedError{Code: code, Message: msg}
	}

	// If syscall mapping isn't meaningful on this platform, keep error string.
	if runtime.GOOS == "windows" && code == "UNKNOWN" {
		return codedError{Code: code, Message: err.Error()}
	}

	return codedError{Code: code, Message: err.Error()}
}
