package test

import (
	"github.com/ethpandaops/minccino/pkg/coordinator/helper"
	human "github.com/ethpandaops/minccino/pkg/coordinator/human-duration"
)

type Config struct {
	Name         string              `yaml:"name" json:"name"`
	Timeout      human.Duration      `yaml:"timeout" json:"timeout"`
	Tasks        []helper.RawMessage `yaml:"tasks" json:"tasks"`
	CleanupTasks []helper.RawMessage `yaml:"cleanupTasks" json:"cleanupTasks"`
}
