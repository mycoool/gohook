package nodeclient

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	ProjectName       string   `json:"projectName"`
	TargetPath        string   `json:"targetPath"`
	Strategy          string   `json:"strategy"`
	IgnoreDefaults    bool     `json:"ignoreDefaults"`
	IgnorePatterns    []string `json:"ignorePatterns"`
	IgnoreFile        string   `json:"ignoreFile"`
	IgnorePermissions bool     `json:"ignorePermissions"`
}

type taskReport struct {
	Status    string `json:"status"`
	Logs      string `json:"logs,omitempty"`
	LastError string `json:"lastError,omitempty"`
}

func (a *Agent) pollAndRunTasks(ctx context.Context) {
	pollInterval := 5 * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task, err := a.pullTask(ctx)
			if err != nil {
				continue
			}
			if task == nil {
				continue
			}
			a.runTask(ctx, task)
		}
	}
}

func (a *Agent) pullTask(ctx context.Context) (*taskResponse, error) {
	endpoint := fmt.Sprintf("%s/sync/nodes/%d/tasks/pull", strings.TrimRight(a.cfg.APIBase, "/"), a.cfg.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Sync-Token", a.cfg.Token)

	resp, err := a.http.Do(req)
	if err != nil {
		log.Printf("nodeclient: pull task failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, fmt.Errorf("pull task unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var task taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (a *Agent) runTask(ctx context.Context, task *taskResponse) {
	var payload taskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: err.Error()})
		return
	}

	if payload.TargetPath == "" || payload.TargetPath == "/" {
		a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: "invalid targetPath"})
		return
	}

	tmpDir := a.cfg.WorkDir
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	work, err := os.MkdirTemp(tmpDir, "gohook-agent-task-")
	if err != nil {
		a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: err.Error()})
		return
	}
	defer os.RemoveAll(work)

	bundlePath := filepath.Join(work, "bundle.tar.gz")
	if err := a.downloadBundle(ctx, task.ID, bundlePath); err != nil {
		a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: err.Error()})
		return
	}

	stage := filepath.Join(work, "stage")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: err.Error()})
		return
	}
	if err := extractBundle(bundlePath, stage, payload.IgnorePermissions); err != nil {
		a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: err.Error()})
		return
	}

	switch strings.ToLower(payload.Strategy) {
	case "overlay":
		if err := overlayApply(stage, payload.TargetPath, payload.IgnorePermissions); err != nil {
			a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: err.Error()})
			return
		}
	default: // mirror
		if err := mirrorSwap(stage, payload.TargetPath); err != nil {
			a.reportTask(ctx, task.ID, taskReport{Status: "failed", LastError: err.Error()})
			return
		}
	}

	a.reportTask(ctx, task.ID, taskReport{Status: "success", Logs: "bundle applied"})
}

func (a *Agent) downloadBundle(ctx context.Context, taskID uint, dst string) error {
	endpoint := fmt.Sprintf("%s/sync/nodes/%d/tasks/%d/bundle", strings.TrimRight(a.cfg.APIBase, "/"), a.cfg.ID, taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Sync-Token", a.cfg.Token)

	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("bundle download status %d: %s", resp.StatusCode, string(body))
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractBundle(path, dst string, ignorePerms bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		rel := filepath.Clean(hdr.Name)
		if strings.HasPrefix(rel, "..") || rel == "." {
			continue
		}
		target := filepath.Join(dst, rel)
		switch hdr.Typeflag {
		case tar.TypeDir:
			mode := os.FileMode(hdr.Mode)
			if ignorePerms {
				mode = 0o755
			}
			if err := os.MkdirAll(target, mode); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode)
			if ignorePerms {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			if !ignorePerms {
				_ = os.Chtimes(target, hdr.ModTime, hdr.ModTime)
			}
		}
	}
	return nil
}

func mirrorSwap(stage, target string) error {
	parent := filepath.Dir(target)
	if parent == "" || parent == "." {
		return fmt.Errorf("invalid target path: %s", target)
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	backup := target + ".bak-" + time.Now().Format("20060102150405")
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(stage, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	if strings.Contains(backup, ".bak-") {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func overlayApply(stage, target string, ignorePerms bool) error {
	return filepath.WalkDir(stage, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(stage, path)
		if rel == "." {
			return nil
		}
		dst := filepath.Join(target, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		mode := fi.Mode()
		if ignorePerms {
			mode = 0o644
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		out.Close()
		return nil
	})
}

func (a *Agent) reportTask(ctx context.Context, taskID uint, report taskReport) {
	endpoint := fmt.Sprintf("%s/sync/nodes/%d/tasks/%d/report", strings.TrimRight(a.cfg.APIBase, "/"), a.cfg.ID, taskID)
	body, _ := json.Marshal(report)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		log.Printf("nodeclient: report task build failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sync-Token", a.cfg.Token)
	resp, err := a.http.Do(req)
	if err != nil {
		log.Printf("nodeclient: report task failed: %v", err)
		return
	}
	resp.Body.Close()
}
