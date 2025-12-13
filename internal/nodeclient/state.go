package nodeclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func LoadState(path string) (State, error) {
	var st State
	b, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, err
	}
	return st, nil
}

func SaveState(path string, st State) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	st.Updated = time.Now().Unix()
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
