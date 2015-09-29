package domain

import (
	"path/filepath"
	"runtime"
	"strconv"
)

type Task struct {
	ApplicationId       string
	DrainUrls           []string
	Index               uint64
	WardenJobId         uint64
	WardenContainerPath string
	SourceName          string
}

func (task *Task) Identifier() string {
	if runtime.GOOS == "windows" {
		return task.WardenContainerPath
	} else {
		return filepath.Join(task.WardenContainerPath, "jobs", strconv.FormatUint(task.WardenJobId, 10))
	}
}
