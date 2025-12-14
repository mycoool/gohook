package nodeclient

// taskResponse mirrors the API response returned by the primary server.
// It is used by the TCP task execution flow.
type taskResponse struct {
	ID          uint   `json:"id"`
	ProjectName string `json:"projectName"`
	NodeID      uint   `json:"nodeId"`
	NodeName    string `json:"nodeName"`
	Driver      string `json:"driver"`
	Status      string `json:"status"`
	Attempt     int    `json:"attempt"`
	Payload     string `json:"payload"`
}

type taskPayload struct {
	ProjectName             string   `json:"projectName"`
	TargetPath              string   `json:"targetPath"`
	Strategy                string   `json:"strategy"`
	IgnoreDefaults          bool     `json:"ignoreDefaults"`
	IgnorePatterns          []string `json:"ignorePatterns"`
	IgnoreFile              string   `json:"ignoreFile"`
	IgnorePermissions       bool     `json:"ignorePermissions"`
	PreserveMode            bool     `json:"preserveMode,omitempty"`
	PreserveMtime           bool     `json:"preserveMtime,omitempty"`
	SymlinkPolicy           string   `json:"symlinkPolicy,omitempty"`
	MirrorFastDelete        bool     `json:"mirrorFastDelete,omitempty"`
	MirrorFastFullscanEvery int      `json:"mirrorFastFullscanEvery,omitempty"`
	MirrorCleanEmptyDirs    bool     `json:"mirrorCleanEmptyDirs,omitempty"`
	MirrorSyncEmptyDirs     bool     `json:"mirrorSyncEmptyDirs,omitempty"`
}
