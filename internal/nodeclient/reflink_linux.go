//go:build linux

package nodeclient

import (
	"os"

	"golang.org/x/sys/unix"
)

func tryCloneFile(dst, src *os.File) error {
	return unix.IoctlFileClone(int(dst.Fd()), int(src.Fd()))
}
