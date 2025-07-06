package models

import "os/exec"

type CommandExecutor func(name string, args ...string) *exec.Cmd
