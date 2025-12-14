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
	return WriteStreamFrame(w, b)
}

func ReadStreamMessage(r io.Reader, dst any) error {
	b, err := ReadStreamFrame(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// WriteStreamFrame writes one length-prefixed frame. Used for both JSON and raw bytes.
func WriteStreamFrame(w io.Writer, payload []byte) error {
	if len(payload) > 16<<20 {
		return fmt.Errorf("frame too large")
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadStreamFrame reads one length-prefixed frame. Used for both JSON and raw bytes.
func ReadStreamFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > 16<<20 {
		return nil, fmt.Errorf("invalid frame size")
	}
	if n == 0 {
		return []byte{}, nil
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}
