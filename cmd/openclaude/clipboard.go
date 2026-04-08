package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// copyToClipboard tries OS clipboard utilities; returns false if none worked.
func copyToClipboard(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	switch runtime.GOOS {
	case "darwin":
		return runClipboardCmd(exec.Command("pbcopy"), text)
	case "linux":
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			if runClipboardCmd(exec.Command("wl-copy"), text) {
				return true
			}
		}
		return runClipboardCmd(exec.Command("xclip", "-selection", "clipboard"), text)
	default:
		return false
	}
}

func runClipboardCmd(cmd *exec.Cmd, text string) bool {
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return false
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err == nil && stderr.Len() == 0
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		return false
	}
}
