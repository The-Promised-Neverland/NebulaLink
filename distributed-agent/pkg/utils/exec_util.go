package utils

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// RunCommand runs a shell command with a timeout (default: 10s).
func RunCommand(name string, args ...string) (string, error) {
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command %s: %w", name, err)
	}
	err := cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("⏱ Command timeout: %s %v", name, args)
		_ = cmd.Process.Kill() // ensure cleanup
		return out.String(), fmt.Errorf("command timed out")
	}
	return out.String(), err
}
