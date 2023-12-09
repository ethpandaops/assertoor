package test

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	human "github.com/ethpandaops/assertoor/pkg/coordinator/human-duration"
)

type Config struct {
	Name         string                 `yaml:"name" json:"name"`
	Disable      bool                   `yaml:"disable" json:"disable"`
	Timeout      human.Duration         `yaml:"timeout" json:"timeout"`
	TestVars     map[string]interface{} `yaml:"testVars" json:"testVars"`
	Tasks        []helper.RawMessage    `yaml:"tasks" json:"tasks"`
	CleanupTasks []helper.RawMessage    `yaml:"cleanupTasks" json:"cleanupTasks"`
}
