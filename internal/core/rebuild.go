package core

import (
	"os/exec"
)

type RebuildResult struct {
	Ok     bool
	Output string
}

func Rebuild(binPath string) RebuildResult {
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/dark")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return RebuildResult{Ok: false, Output: string(out)}
	}
	return RebuildResult{Ok: true}
}
