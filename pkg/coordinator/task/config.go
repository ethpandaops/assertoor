package task

import (
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/helper"
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/human-duration"
)

type Config struct {
	// The name of the task to run.
	Name string `yaml:"name" json:"name"`
	// The configuration object of the task.
	Config *helper.RawMessage `yaml:"config" json:"config"`
	// The title of the task - this is used to describe the task to the user.
	Title string `yaml:"title" json:"title"`
	// Timeout defines the max time waiting for the condition to be met.
	Timeout human.Duration `yaml:"timeout" json:"timeout"`
}
