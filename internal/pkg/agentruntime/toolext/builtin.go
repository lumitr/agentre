package toolext

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	RegisterBuiltin("web_search", &webSearchExecutor{})
	RegisterBuiltin("code_search", &codeSearchExecutor{})
	RegisterBuiltin("read_file", &readFileExecutor{})
	RegisterBuiltin("list_files", &listFilesExecutor{})
}

// --- web_search ---

type webSearchParams struct {
	Query string `json:"query"`
}

type webSearchExecutor struct{}

func (e *webSearchExecutor) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p webSearchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("web_search: invalid params: %w", err)
	}
	return fmt.Sprintf("web_search: query=%q (search API integration pending)", p.Query), nil
}

// --- code_search ---

type codeSearchParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

type codeSearchExecutor struct{}

func (e *codeSearchExecutor) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p codeSearchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("code_search: invalid params: %w", err)
	}
	if p.Pattern == "" {
		return "", fmt.Errorf("code_search: pattern is required")
	}
	args := []string{"-rn", "--color=never", p.Pattern}
	if p.Path != "" {
		args = append(args, p.Path)
	}
	out, err := exec.Command("grep", args...).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "no matches found", nil
		}
		return "", fmt.Errorf("code_search: %w", err)
	}
	result := string(out)
	if len(result) > 64*1024 {
		result = result[:64*1024] + "\n... (truncated)"
	}
	return result, nil
}

// --- read_file ---

type readFileParams struct {
	Path string `json:"path"`
}

type readFileExecutor struct{}

func (e *readFileExecutor) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p readFileParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("read_file: invalid params: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("read_file: path is required")
	}
	cleaned := filepath.Clean(p.Path)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("read_file: path traversal not allowed")
	}
	out, err := exec.Command("cat", "-n", cleaned).Output()
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	result := string(out)
	if len(result) > 64*1024 {
		result = result[:64*1024] + "\n... (truncated)"
	}
	return result, nil
}

// --- list_files ---

type listFilesParams struct {
	Path string `json:"path,omitempty"`
}

type listFilesExecutor struct{}

func (e *listFilesExecutor) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p listFilesParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("list_files: invalid params: %w", err)
	}
	dir := p.Path
	if dir == "" {
		dir = "."
	}
	// Try tree first, fall back to ls.
	out, err := exec.Command("tree", dir).Output()
	if err != nil {
		out, err = exec.Command("ls", "-la", dir).Output()
		if err != nil {
			return "", fmt.Errorf("list_files: %w", err)
		}
	}
	result := string(out)
	if len(result) > 64*1024 {
		result = result[:64*1024] + "\n... (truncated)"
	}
	return result, nil
}
