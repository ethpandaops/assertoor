package test

import "github.com/samcm/sync-test-coordinator/pkg/coordinator/helper"

type TaskConfig struct {
	Name   string             `yaml:"name"`
	Config *helper.RawMessage `yaml:"config"`
}

type Config struct {
	Name  string       `yaml:"name"`
	Tasks []TaskConfig `yaml:"tasks"`
}
