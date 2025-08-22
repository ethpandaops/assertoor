package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Descriptor struct {
	id       string
	source   string
	basePath string
	config   *types.TestConfig
	vars     types.Variables
	err      error
}

func NewDescriptor(testID, testSrc, basePath string, config *types.TestConfig, variables types.Variables) *Descriptor {
	return &Descriptor{
		id:       testID,
		source:   testSrc,
		basePath: basePath,
		config:   config,
		vars:     variables,
	}
}

func LoadTestDescriptors(ctx context.Context, globalVars types.Variables, localTests []*types.TestConfig, externalTests []*types.ExternalTestConfig) []types.TestDescriptor {
	descriptors := []types.TestDescriptor{}

	workingDir, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Warn("failed to get working directory")
	}

	// load local tests
	for testIdx, testCfg := range localTests {
		testID := testCfg.ID
		testSrc := fmt.Sprintf("local-%v", testIdx+1)

		if testID == "" {
			testID = testSrc
		}

		testVars := globalVars.NewScope()

		for k, v := range testCfg.Config {
			testVars.SetVar(k, v)
		}

		err := testVars.CopyVars(globalVars, testCfg.ConfigVars)

		descriptors = append(descriptors, &Descriptor{
			id:       testID,
			source:   testSrc,
			basePath: workingDir,
			vars:     testVars,
			config:   localTests[testIdx],
			err:      err,
		})
	}

	// load external tests
	for testIdx, extTestCfg := range externalTests {
		testSrc := fmt.Sprintf("external:%v", extTestCfg.File)
		testID := ""

		testConfig, testVars, basePath, err := LoadExternalTestConfig(ctx, globalVars, extTestCfg)

		if testConfig != nil && testConfig.ID != "" {
			testID = testConfig.ID
		}

		if testID == "" {
			testID = fmt.Sprintf("external-%v", testIdx)
		}

		descriptors = append(descriptors, &Descriptor{
			id:       testID,
			source:   testSrc,
			basePath: basePath,
			config:   testConfig,
			vars:     testVars,
			err:      err,
		})
	}

	return descriptors
}

func LoadExternalTestConfig(ctx context.Context, globalVars types.Variables, extTestCfg *types.ExternalTestConfig) (*types.TestConfig, types.Variables, string, error) {
	var reader io.Reader
	var basePath string

	if strings.HasPrefix(extTestCfg.File, "http://") || strings.HasPrefix(extTestCfg.File, "https://") {
		parsedURL, err := url.Parse(extTestCfg.File)
		if err != nil {
			return nil, nil, "", err
		}

		// Remove the filename from the path
		parsedURL.Path = path.Dir(parsedURL.Path)
		parsedURL.RawQuery = ""
		parsedURL.Fragment = ""

		basePath = parsedURL.String()

		client := &http.Client{Timeout: time.Second * 120}

		req, err := http.NewRequestWithContext(ctx, "GET", extTestCfg.File, http.NoBody)
		if err != nil {
			return nil, nil, basePath, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, basePath, err
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				logrus.WithError(err).Warn("failed to close response body")
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return nil, nil, basePath, fmt.Errorf("error loading test config from url: %v, result: %v %v", extTestCfg.File, resp.StatusCode, resp.Status)
		}

		reader = resp.Body
	} else {
		basePath = path.Dir(extTestCfg.File)

		f, err := os.Open(extTestCfg.File)
		if err != nil {
			return nil, nil, basePath, fmt.Errorf("error loading test config from file %v: %w", extTestCfg.File, err)
		}

		defer func() {
			if err := f.Close(); err != nil {
				logrus.WithError(err).Warn("failed to close file")
			}
		}()

		reader = f
	}

	decoder := yaml.NewDecoder(reader)
	testConfig := &types.TestConfig{}
	testVars := globalVars.NewScope()

	err := decoder.Decode(testConfig)
	if err != nil {
		return nil, nil, basePath, fmt.Errorf("error decoding external test config %v: %v", extTestCfg.File, err)
	}

	if testConfig.Config == nil {
		testConfig.Config = map[string]interface{}{}
	}

	if testConfig.ConfigVars == nil {
		testConfig.ConfigVars = map[string]string{}
	}

	for k, v := range testConfig.Config {
		testVars.SetDefaultVar(k, v)
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

	if extTestCfg.Schedule != nil {
		testConfig.Schedule = extTestCfg.Schedule
	}

	for k, v := range extTestCfg.Config {
		testConfig.Config[k] = v
		testVars.SetVar(k, v)
	}

	for k, v := range extTestCfg.ConfigVars {
		testConfig.ConfigVars[k] = v
	}

	err = testVars.CopyVars(testVars, testConfig.ConfigVars)
	if err != nil {
		return nil, nil, basePath, fmt.Errorf("error decoding external test configVars %v: %v", extTestCfg.File, err)
	}

	return testConfig, testVars, basePath, nil
}

func (d *Descriptor) ID() string {
	return d.id
}

func (d *Descriptor) Source() string {
	return d.source
}

func (d *Descriptor) BasePath() string {
	return d.basePath
}

func (d *Descriptor) Config() *types.TestConfig {
	return d.config
}

func (d *Descriptor) Vars() types.Variables {
	return d.vars
}

func (d *Descriptor) Err() error {
	return d.err
}
