package syncnode

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Stream protocol: uint32(be) length + JSON payload.

type streamMessage struct {
	Type string `json:"type"`
}

func WriteStreamMessage(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) > 16<<20 {
		return fmt.Errorf("message too large")
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func ReadStreamMessage(r io.Reader, dst any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > 16<<20 {
		return fmt.Errorf("invalid message size")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
