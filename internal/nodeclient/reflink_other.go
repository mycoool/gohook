//go:build !linux

package nodeclient

import (
	"errors"
	"os"
)

func tryCloneFile(_ *os.File, _ *os.File) error {
	return errors.New("reflink not supported")
}
