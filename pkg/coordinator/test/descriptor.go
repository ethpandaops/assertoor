package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"gopkg.in/yaml.v3"
)

type Descriptor struct {
	id     string
	source string
	config *types.TestConfig
	err    error
}

func NewDescriptor(testID, testSrc string, config *types.TestConfig) *Descriptor {
	return &Descriptor{
		id:     testID,
		source: testSrc,
		config: config,
	}
}

func LoadTestDescriptors(ctx context.Context, localTests []*types.TestConfig, externalTests []*types.ExternalTestConfig) []types.TestDescriptor {
	descriptors := []types.TestDescriptor{}

	// load local tests
	for testIdx, testCfg := range localTests {
		testID := testCfg.ID
		testSrc := fmt.Sprintf("local-%v", testIdx+1)

		if testID == "" {
			testID = testSrc
		}

		descriptors = append(descriptors, &Descriptor{
			id:     testID,
			source: testSrc,
			config: localTests[testIdx],
		})
	}

	// load external tests
	for testIdx, extTestCfg := range externalTests {
		testSrc := fmt.Sprintf("external:%v", extTestCfg.File)
		testID := ""

		testConfig, err := LoadExternalTestConfig(ctx, extTestCfg)

		if testConfig.ID != "" {
			testID = testConfig.ID
		}

		if testID == "" {
			testID = fmt.Sprintf("external-%v", testIdx)
		}

		descriptors = append(descriptors, &Descriptor{
			id:     testID,
			source: testSrc,
			config: testConfig,
			err:    err,
		})
	}

	return descriptors
}

func LoadExternalTestConfig(ctx context.Context, extTestCfg *types.ExternalTestConfig) (*types.TestConfig, error) {
	var reader io.Reader

	if strings.HasPrefix(extTestCfg.File, "http://") || strings.HasPrefix(extTestCfg.File, "https://") {
		client := &http.Client{Timeout: time.Second * 120}

		req, err := http.NewRequestWithContext(ctx, "GET", extTestCfg.File, http.NoBody)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error loading test config from url: %v, result: %v %v", extTestCfg.File, resp.StatusCode, resp.Status)
		}

		reader = resp.Body
	} else {
		f, err := os.Open(extTestCfg.File)
		if err != nil {
			return nil, fmt.Errorf("error loading test config from file %v: %w", extTestCfg.File, err)
		}

		defer f.Close()

		reader = f
	}

	decoder := yaml.NewDecoder(reader)
	testConfig := &types.TestConfig{}

	err := decoder.Decode(testConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding external test config %v: %v", extTestCfg.File, err)
	}

	if testConfig.Config == nil {
		testConfig.Config = map[string]interface{}{}
	}

	if testConfig.ConfigVars == nil {
		testConfig.ConfigVars = map[string]string{}
	}

	if extTestCfg.ID != "" {
		testConfig.ID = extTestCfg.ID
	}

	if extTestCfg.Name != "" {
		testConfig.Name = extTestCfg.Name
	}

	if extTestCfg.Timeout != nil {
		testConfig.Timeout = *extTestCfg.Timeout
	}

	for k, v := range extTestCfg.Config {
		testConfig.Config[k] = v
	}

	for k, v := range extTestCfg.ConfigVars {
		testConfig.ConfigVars[k] = v
	}

	return testConfig, nil
}

func (d *Descriptor) ID() string {
	return d.id
}

func (d *Descriptor) Source() string {
	return d.source
}

func (d *Descriptor) Config() *types.TestConfig {
	return d.config
}

func (d *Descriptor) Err() error {
	return d.err
}
