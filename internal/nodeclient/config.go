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
