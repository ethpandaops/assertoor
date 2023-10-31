package test

import (
	human "github.com/samcm/sync-test-coordinator/pkg/coordinator/human-duration"
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
)

type Config struct {
	Name         string         `yaml:"name" json:"name"`
	Timeout      human.Duration `yaml:"timeout" json:"timeout"`
	Tasks        []task.Config  `yaml:"tasks" json:"tasks"`
	CleanupTasks []task.Config  `yaml:"cleanup_tasks" json:"cleanup_tasks"`
}
