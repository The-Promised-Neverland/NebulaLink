package utils

import (
    "bytes"
    "log"
    "os/exec"
    "runtime"
    "syscall"
    "time"
)

// executes commands and timeouts safely from process (if hangs)
func RunCommand(name string, args ...string) (string, error) {
    cmd := exec.Command(name, args...)
    if runtime.GOOS == "windows" {
        cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    }
    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &out
    if err := cmd.Start(); err != nil {
        return "", err
    }
    done := make(chan error)
    go func() { done <- cmd.Wait() }()
    select {
    case err := <-done:
        return out.String(), err
    case <-time.After(10 * time.Second):
        log.Println("⏱ Command timeout:", name)
        _ = cmd.Process.Kill()
        return "", nil
    }
}
