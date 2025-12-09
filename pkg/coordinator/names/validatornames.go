package names

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type ValidatorNames struct {
	config     *Config
	logger     logrus.FieldLogger
	namesMutex sync.RWMutex
	names      map[uint64]string
}

func NewValidatorNames(config *Config, logger logrus.FieldLogger) *ValidatorNames {
	return &ValidatorNames{
		config: config,
		logger: logger.WithField("module", "names"),
	}
}

func (vn *ValidatorNames) GetValidatorName(index uint64) string {
	if !vn.namesMutex.TryRLock() {
		return ""
	}
	defer vn.namesMutex.RUnlock()

	if vn.names == nil {
		return ""
	}

	return vn.names[index]
}

func (vn *ValidatorNames) LoadValidatorNames() {
	vn.namesMutex.Lock()
	defer vn.namesMutex.Unlock()

	vn.names = make(map[uint64]string)

	if vn.config == nil {
		return
	}

	// load names
	if vn.config.InventoryYaml != "" {
		err := vn.loadFromYaml(vn.config.InventoryYaml)
		if err != nil {
			vn.logger.WithError(err).Errorf("error while loading validator names from yaml")
		}
	}

	if vn.config.InventoryURL != "" {
		err := vn.loadFromRangesAPI(vn.config.InventoryURL)
		if err != nil {
			vn.logger.WithError(err).Errorf("error while loading validator names inventory")
		}
	}

	if vn.config.Inventory != nil {
		nameCount := vn.parseNamesMap(vn.config.Inventory)
		if nameCount > 0 {
			vn.logger.Infof("loaded %v validator names from config", nameCount)
		}
	}
}

func (vn *ValidatorNames) loadFromYaml(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("error opening validator names file %v: %v", fileName, err)
	}

	defer func() {
		if err2 := f.Close(); err2 != nil {
			vn.logger.WithError(err2).Warn("failed to close file")
		}
	}()

	namesYaml := map[string]string{}
	decoder := yaml.NewDecoder(f)

	err = decoder.Decode(&namesYaml)
	if err != nil {
		return fmt.Errorf("error decoding validator names file %v: %v", fileName, err)
	}

	nameCount := vn.parseNamesMap(namesYaml)
	vn.logger.Infof("loaded %v validator names from yaml (%v)", nameCount, fileName)

	return nil
}

func (vn *ValidatorNames) parseNamesMap(names map[string]string) int {
	nameCount := 0

	for idxStr, name := range names {
		rangeParts := strings.Split(idxStr, "-")

		minIdx, err := strconv.ParseUint(rangeParts[0], 10, 64)
		if err != nil {
			continue
		}

		maxIdx := minIdx + 1
		if len(rangeParts) > 1 {
			maxIdx, err = strconv.ParseUint(rangeParts[1], 10, 64)
			if err != nil {
				continue
			}
		}

		for idx := minIdx; idx <= maxIdx; idx++ {
			vn.names[idx] = name
			nameCount++
		}
	}

	return nameCount
}

type validatorNamesRangesResponse struct {
	Ranges map[string]string `json:"ranges"`
}

func (vn *ValidatorNames) loadFromRangesAPI(apiURL string) error {
	vn.logger.Debugf("Loading validator names from inventory: %v", apiURL)

	client := &http.Client{Timeout: time.Second * 120}

	resp, err := client.Get(apiURL)
	if err != nil {
		return fmt.Errorf("could not fetch inventory (%v): %v", getRedactedURL(apiURL), err)
	}

	defer func() {
		if err2 := resp.Body.Close(); err2 != nil {
			vn.logger.WithError(err2).Warn("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			vn.logger.Errorf("could not fetch inventory (%v): not found", getRedactedURL(apiURL))
			return nil
		}

		data, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("url: %v, error-response: %s", getRedactedURL(apiURL), data)
	}

	rangesResponse := &validatorNamesRangesResponse{}
	dec := json.NewDecoder(resp.Body)

	err = dec.Decode(&rangesResponse)
	if err != nil {
		return fmt.Errorf("error parsing validator ranges response: %v", err)
	}

	nameCount := 0

	for rangeStr, name := range rangesResponse.Ranges {
		rangeParts := strings.Split(rangeStr, "-")

		minIdx, err := strconv.ParseUint(rangeParts[0], 10, 64)
		if err != nil {
			continue
		}

		maxIdx := minIdx + 1
		if len(rangeParts) > 1 {
			maxIdx, err = strconv.ParseUint(rangeParts[1], 10, 64)
			if err != nil {
				continue
			}
		}

		for idx := minIdx; idx <= maxIdx; idx++ {
			vn.names[idx] = name
			nameCount++
		}
	}

	vn.logger.Infof("loaded %v validator names from inventory api (%v)", nameCount, getRedactedURL(apiURL))

	return nil
}

func getRedactedURL(requrl string) string {
	urlData, _ := url.Parse(requrl)
	logurl := ""

	if urlData != nil {
		logurl = urlData.Redacted()
	} else {
		logurl = requrl
	}

	return logurl
}
