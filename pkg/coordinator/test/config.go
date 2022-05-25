package test

import "github.com/samcm/sync-test-coordinator/pkg/coordinator/helper"

type TaskConfig struct {
	Name   string             `yaml:"name" json:"name"`
	Config *helper.RawMessage `yaml:"config" json:"config"`
}

type Config struct {
	Name  string       `yaml:"name" json:"name"`
	Tasks []TaskConfig `yaml:"tasks" json:"tasks"`
}
