package nodeclient

import (
	"fmt"
	"io"
	"os"
)

func cloneOrCopyFile(dst, src *os.File) error {
	if dst == nil || src == nil {
		return fmt.Errorf("invalid file handle")
	}
	if err := tryCloneFile(dst, src); err == nil {
		return nil
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := dst.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := dst.Truncate(0); err != nil {
		return err
	}
	buf := make([]byte, 4<<20)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		return err
	}
	return nil
}
