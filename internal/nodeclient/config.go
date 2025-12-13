package nodeclient

import "time"

// RuntimeConfig holds agent settings after parsing env/flags.
type RuntimeConfig struct {
	NodeID   uint
	APIBase  string
	Token    string
	Interval time.Duration
	NodeName string
	Version  string
	WorkDir  string
}

// State is persisted to disk under DataDir to simplify subsequent agent restarts.
type State struct {
	NodeID  uint   `json:"nodeId"`
	Token   string `json:"token,omitempty"`
	Server  string `json:"server,omitempty"`
	Updated int64  `json:"updated,omitempty"`
}
