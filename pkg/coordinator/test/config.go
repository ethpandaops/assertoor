package test

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	human "github.com/ethpandaops/assertoor/pkg/coordinator/human-duration"
)

type Config struct {
	Name         string                 `yaml:"name" json:"name"`
	Disable      bool                   `yaml:"disable" json:"disable"`
	Timeout      human.Duration         `yaml:"timeout" json:"timeout"`
	Config       map[string]interface{} `yaml:"config" json:"config"`
	ConfigVars   map[string]string      `yaml:"configVars" json:"configVars"`
	Tasks        []helper.RawMessage    `yaml:"tasks" json:"tasks"`
	CleanupTasks []helper.RawMessage    `yaml:"cleanupTasks" json:"cleanupTasks"`
}

type ExternalConfig struct {
	File       string                 `yaml:"file" json:"file"`
	Name       string                 `yaml:"name" json:"name"`
	Timeout    *human.Duration        `yaml:"timeout" json:"timeout"`
	Config     map[string]interface{} `yaml:"config" json:"config"`
	ConfigVars map[string]string      `yaml:"configVars" json:"configVars"`
}
