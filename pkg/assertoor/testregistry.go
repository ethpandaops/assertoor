package assertoor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/test"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

type TestRegistry struct {
	coordinator types.Coordinator

	testDescriptors      map[string]testDescriptorEntry
	testDescriptorsMutex sync.RWMutex
	testDescriptorIndex  uint64
}

type testDescriptorEntry struct {
	descriptor types.TestDescriptor
	index      uint64
}

func NewTestRegistry(coordinator types.Coordinator) *TestRegistry {
	return &TestRegistry{
		coordinator: coordinator,

		testDescriptors: map[string]testDescriptorEntry{},
	}
}

func (c *TestRegistry) GetTestDescriptors() []types.TestDescriptor {
	c.testDescriptorsMutex.RLock()
	defer c.testDescriptorsMutex.RUnlock()

	descriptors := make([]types.TestDescriptor, len(c.testDescriptors))
	idx := 0

	for _, descriptorEntry := range c.testDescriptors {
		descriptors[idx] = descriptorEntry.descriptor
		idx++
	}

	sort.Slice(descriptors, func(a, b int) bool {
		entryA := c.testDescriptors[descriptors[a].ID()]
		entryB := c.testDescriptors[descriptors[b].ID()]

		return entryA.index < entryB.index
	})

	return descriptors
}

func (c *TestRegistry) LoadTests(ctx context.Context, local []*types.TestConfig, external []*types.ExternalTestConfig) {
	dbTestConfigs, err := c.coordinator.Database().GetTestConfigs()
	if err != nil {
		c.coordinator.Logger().Errorf("error loading test configs from db: %v", err)
	}

	externalTests := []*types.ExternalTestConfig{}

	for _, dbTestConfig := range dbTestConfigs {
		externalTest := &types.ExternalTestConfig{
			ID:         dbTestConfig.TestID,
			File:       dbTestConfig.Source,
			Name:       dbTestConfig.Name,
			Config:     map[string]interface{}{},
			ConfigVars: map[string]string{},
			Schedule: &types.TestSchedule{
				Startup: false,
				Cron:    []string{},
			},
			YamlSource: dbTestConfig.YamlSource,
		}

		if dbTestConfig.Timeout > 0 {
			externalTest.Timeout = &helper.Duration{Duration: time.Duration(dbTestConfig.Timeout) * time.Second}
		}

		// When YamlSource is present, config/configVars are already in the YAML,
		// so don't load them from DB (they would be treated as overrides)
		if dbTestConfig.YamlSource == "" {
			if err := yaml.Unmarshal([]byte(dbTestConfig.Config), &externalTest.Config); err != nil {
				c.coordinator.Logger().Errorf("error decoding test config %v from db: %v", dbTestConfig.TestID, err)
				continue
			}

			if err := yaml.Unmarshal([]byte(dbTestConfig.ConfigVars), &externalTest.ConfigVars); err != nil {
				c.coordinator.Logger().Errorf("error decoding test configVars %v from db: %v", dbTestConfig.TestID, err)
				continue
			}
		}

		if dbTestConfig.ScheduleCronYaml != "" {
			if err := yaml.Unmarshal([]byte(dbTestConfig.ScheduleCronYaml), &externalTest.Schedule.Cron); err != nil {
				c.coordinator.Logger().Errorf("error decoding test cron schedule %v from db: %v", dbTestConfig.TestID, err)
				continue
			}
		}

		externalTests = append(externalTests, externalTest)
	}

	descriptors := test.LoadTestDescriptors(ctx, c.coordinator.GlobalVariables(), local, externalTests)
	newCfgTests := []*db.TestConfig{}

	for _, cfgExternalTest := range external {
		found := false

		for _, externalTest := range externalTests {
			if externalTest.ID != cfgExternalTest.ID && externalTest.File != cfgExternalTest.File {
				continue
			}

			cfgExternalTest.Config = externalTest.Config
			cfgExternalTest.ConfigVars = externalTest.ConfigVars
			cfgExternalTest.Schedule = externalTest.Schedule
			found = true

			break
		}

		dbTestCfg, err := c.externalTestCfgToDB(cfgExternalTest, "")
		if err != nil {
			c.coordinator.Logger().Errorf("error converting external test config %v to db: %v", cfgExternalTest.ID, err)
		}

		if found {
			continue
		}

		testDescriptor, err := c.AddExternalTest(ctx, cfgExternalTest)
		if err != nil {
			c.coordinator.Logger().Errorf("error adding external test %v: %v", cfgExternalTest.ID, err)
			continue
		}

		dbTestCfg.TestID = testDescriptor.ID()
		dbTestCfg.Name = testDescriptor.Config().Name

		newCfgTests = append(newCfgTests, dbTestCfg)
		descriptors = append(descriptors, testDescriptor)
	}

	if len(newCfgTests) > 0 {
		err := c.coordinator.Database().RunTransaction(func(tx *sqlx.Tx) error {
			for _, dbTestCfg := range newCfgTests {
				err := c.coordinator.Database().InsertTestConfig(tx, dbTestCfg)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			c.coordinator.Logger().Errorf("error adding new test configs to db: %v", err)
		}
	}

	errCount := 0

	c.testDescriptorsMutex.Lock()
	defer c.testDescriptorsMutex.Unlock()

	c.testDescriptors = map[string]testDescriptorEntry{}

	for _, descriptor := range descriptors {
		if descriptor.Err() != nil {
			c.coordinator.Logger().Errorf("error while loading test '%v': %v", descriptor.ID(), descriptor.Err())

			errCount++
		}

		c.testDescriptorIndex++
		entryIndex := c.testDescriptorIndex

		c.testDescriptors[descriptor.ID()] = testDescriptorEntry{
			descriptor: descriptor,
			index:      entryIndex,
		}
	}

	c.coordinator.Logger().Infof("loaded %v test descriptors (%v errors)", len(descriptors), errCount)
}

func (c *TestRegistry) AddLocalTest(testConfig *types.TestConfig) (types.TestDescriptor, error) {
	return c.AddLocalTestWithYaml(testConfig, "")
}

func (c *TestRegistry) AddLocalTestWithYaml(testConfig *types.TestConfig, yamlSource string) (types.TestDescriptor, error) {
	if testConfig.ID == "" {
		return nil, fmt.Errorf("cannot add test descriptor without ID")
	}

	testVars := vars.NewVariables(c.coordinator.GlobalVariables())

	for k, v := range testConfig.Config {
		testVars.SetDefaultVar(k, v)
	}

	err := testVars.CopyVars(testVars, testConfig.ConfigVars)
	if err != nil {
		return nil, fmt.Errorf("failed decoding configVars: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed getting working directory: %v", err)
	}

	testDescriptor := test.NewDescriptor(testConfig.ID, "api-call", workingDir, testConfig, testVars)

	// Persist to database
	dbTestCfg, err := c.localTestCfgToDB(testConfig, yamlSource)
	if err != nil {
		return nil, fmt.Errorf("error converting local test config for db: %v", err)
	}

	err = c.coordinator.Database().RunTransaction(func(tx *sqlx.Tx) error {
		return c.coordinator.Database().InsertTestConfig(tx, dbTestCfg)
	})
	if err != nil {
		c.coordinator.Logger().Errorf("error persisting local test to db: %v", err)
	}

	c.testDescriptorsMutex.Lock()
	defer c.testDescriptorsMutex.Unlock()

	entryIndex := c.testDescriptors[testDescriptor.ID()].index
	if entryIndex == 0 {
		c.testDescriptorIndex++
		entryIndex = c.testDescriptorIndex
	}

	c.testDescriptors[testDescriptor.ID()] = testDescriptorEntry{
		descriptor: testDescriptor,
		index:      entryIndex,
	}

	return testDescriptor, nil
}

func (c *TestRegistry) localTestCfgToDB(testConfig *types.TestConfig, yamlSource string) (*db.TestConfig, error) {
	dbTestCfg := &db.TestConfig{
		TestID: testConfig.ID,
		Source: "api-call",
		Name:   testConfig.Name,
	}

	if testConfig.Timeout.Duration > 0 {
		dbTestCfg.Timeout = int(testConfig.Timeout.Seconds())
	}

	if testConfig.Schedule != nil {
		dbTestCfg.ScheduleStartup = testConfig.Schedule.Startup

		if len(testConfig.Schedule.Cron) > 0 {
			cronYaml, err := yaml.Marshal(testConfig.Schedule.Cron)
			if err != nil {
				return nil, fmt.Errorf("error encoding test cron schedule: %v", err)
			}

			dbTestCfg.ScheduleCronYaml = string(cronYaml)
		}
	}

	// When yamlSource is provided, config/configVars are already in the YAML,
	// so don't store them separately (they would be treated as overrides on reload)
	if yamlSource == "" {
		configYaml, err := yaml.Marshal(testConfig.Config)
		if err != nil {
			return nil, fmt.Errorf("error encoding test config: %v", err)
		}

		dbTestCfg.Config = string(configYaml)

		configVarsYaml, err := yaml.Marshal(testConfig.ConfigVars)
		if err != nil {
			return nil, fmt.Errorf("error encoding test configVars: %v", err)
		}

		dbTestCfg.ConfigVars = string(configVarsYaml)
	}

	dbTestCfg.YamlSource = yamlSource

	return dbTestCfg, nil
}

func (c *TestRegistry) AddExternalTest(ctx context.Context, extTestCfg *types.ExternalTestConfig) (types.TestDescriptor, error) {
	testConfig, testVars, basePath, yamlSource, err := test.LoadExternalTestConfig(ctx, c.coordinator.GlobalVariables(), extTestCfg)
	if err != nil {
		return nil, fmt.Errorf("failed loading test config from %v: %w", extTestCfg.File, err)
	}

	if testConfig.ID == "" {
		return nil, errors.New("test id missing or empty")
	}

	if testConfig.Name == "" {
		return nil, errors.New("test name missing or empty")
	}

	if len(testConfig.Tasks) == 0 {
		return nil, errors.New("test must have 1 or more tasks")
	}

	testDescriptor := test.NewDescriptor(testConfig.ID, fmt.Sprintf("external:%v", extTestCfg.File), basePath, testConfig, testVars)
	extTestCfg.ID = testDescriptor.ID()
	extTestCfg.Name = testConfig.Name

	dbTestCfg, err := c.externalTestCfgToDB(extTestCfg, yamlSource)
	if err != nil {
		return nil, fmt.Errorf("error converting external test config %v for db: %v", extTestCfg.ID, err)
	}

	err = c.coordinator.Database().RunTransaction(func(tx *sqlx.Tx) error {
		return c.coordinator.Database().InsertTestConfig(tx, dbTestCfg)
	})
	if err != nil {
		c.coordinator.Logger().Errorf("error adding new test configs to db: %v", err)
	}

	c.testDescriptorsMutex.Lock()
	defer c.testDescriptorsMutex.Unlock()

	entryIndex := c.testDescriptors[testDescriptor.ID()].index
	if entryIndex == 0 {
		c.testDescriptorIndex++
		entryIndex = c.testDescriptorIndex
	}

	c.testDescriptors[testDescriptor.ID()] = testDescriptorEntry{
		descriptor: testDescriptor,
		index:      entryIndex,
	}

	return testDescriptor, nil
}

func (c *TestRegistry) externalTestCfgToDB(cfgExternalTest *types.ExternalTestConfig, yamlSource string) (*db.TestConfig, error) {
	dbTestCfg := &db.TestConfig{
		TestID:     cfgExternalTest.ID,
		Source:     cfgExternalTest.File,
		Name:       cfgExternalTest.Name,
		YamlSource: yamlSource,
	}

	if cfgExternalTest.Timeout != nil {
		dbTestCfg.Timeout = int(cfgExternalTest.Timeout.Seconds())
	}

	if cfgExternalTest.Schedule != nil {
		dbTestCfg.ScheduleStartup = cfgExternalTest.Schedule.Startup

		if len(cfgExternalTest.Schedule.Cron) > 0 {
			cronYaml, err := yaml.Marshal(cfgExternalTest.Schedule.Cron)
			if err != nil {
				return nil, fmt.Errorf("error encoding test cron schedule %v: %v", cfgExternalTest.ID, err)
			}

			dbTestCfg.ScheduleCronYaml = string(cronYaml)
		}
	} else {
		dbTestCfg.ScheduleStartup = true
	}

	configYaml, err := yaml.Marshal(cfgExternalTest.Config)
	if err != nil {
		return nil, fmt.Errorf("error encoding test config %v: %v", cfgExternalTest.ID, err)
	}

	dbTestCfg.Config = string(configYaml)

	configVarsYaml, err := yaml.Marshal(cfgExternalTest.ConfigVars)
	if err != nil {
		return nil, fmt.Errorf("error encoding test configVars %v: %v", cfgExternalTest.ID, err)
	}

	dbTestCfg.ConfigVars = string(configVarsYaml)

	return dbTestCfg, nil
}

func (c *TestRegistry) DeleteTest(testID string) error {
	c.testDescriptorsMutex.Lock()

	if _, ok := c.testDescriptors[testID]; !ok {
		c.testDescriptorsMutex.Unlock()
		return nil
	}

	delete(c.testDescriptors, testID)

	c.testDescriptorsMutex.Unlock()

	return c.coordinator.Database().RunTransaction(func(tx *sqlx.Tx) error {
		return c.coordinator.Database().DeleteTestConfig(tx, testID)
	})
}
