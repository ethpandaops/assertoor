package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"gopkg.in/yaml.v3"
)

type PostTestsRegisterUploadResponse struct {
	TestID string         `json:"test_id"`
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

// PostTestsRegisterUpload godoc
// @Id postTestsRegisterUpload
// @Summary Register new test via uploaded YAML file
// @Tags Test
// @Description Upload a YAML test configuration file and register the test. Returns the test id and name of the added test.
// @Produce json
// @Accept multipart/form-data
// @Param playbook formData file true "YAML test configuration file"
// @Param name formData string false "Custom test name override"
// @Param timeout formData integer false "Custom timeout in seconds"
// @Param config formData string false "Custom config overrides as YAML"
// @Param configVars formData string false "Custom config variables as YAML"
// @Success 200 {object} Response{data=PostTestsRegisterUploadResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/tests/register_upload [post]
func (ah *APIHandler) PostTestsRegisterUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error parsing multipart form: %v", err), http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, fileHeader, err := r.FormFile("playbook")
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error retrieving uploaded file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error reading uploaded file: %v", err), http.StatusBadRequest)
		return
	}

	// Parse YAML test configuration
	var testConfig types.TestConfig
	err = yaml.Unmarshal(fileContent, &testConfig)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error parsing YAML test configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if testConfig.ID == "" {
		ah.sendErrorResponse(w, r.URL.String(), "test id missing or empty in uploaded file", http.StatusBadRequest)
		return
	}

	if testConfig.Name == "" {
		ah.sendErrorResponse(w, r.URL.String(), "test name missing or empty in uploaded file", http.StatusBadRequest)
		return
	}

	if len(testConfig.Tasks) == 0 {
		ah.sendErrorResponse(w, r.URL.String(), "test must have 1 or more tasks", http.StatusBadRequest)
		return
	}

	// Apply form overrides
	customName := r.FormValue("name")
	if customName != "" {
		testConfig.Name = customName
	}

	customTimeout := r.FormValue("timeout")
	if customTimeout != "" {
		timeoutDuration, err := time.ParseDuration(customTimeout + "s")
		if err != nil {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("invalid timeout format: %v", err), http.StatusBadRequest)
			return
		}
		testConfig.Timeout = helper.Duration{Duration: timeoutDuration}
	}

	// Parse custom config overrides
	customConfig := r.FormValue("config")
	if customConfig != "" {
		var configOverrides map[string]interface{}
		err = yaml.Unmarshal([]byte(customConfig), &configOverrides)
		if err != nil {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error parsing config overrides: %v", err), http.StatusBadRequest)
			return
		}
		// Merge with existing config
		if testConfig.Config == nil {
			testConfig.Config = make(map[string]interface{})
		}
		for key, value := range configOverrides {
			testConfig.Config[key] = value
		}
	}

	// Parse custom config variables
	customConfigVars := r.FormValue("configVars")
	if customConfigVars != "" {
		var configVars map[string]string
		err = yaml.Unmarshal([]byte(customConfigVars), &configVars)
		if err != nil {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error parsing config variables: %v", err), http.StatusBadRequest)
			return
		}
		// Merge with existing config vars
		if testConfig.ConfigVars == nil {
			testConfig.ConfigVars = make(map[string]string)
		}
		for key, value := range configVars {
			testConfig.ConfigVars[key] = value
		}
	}

	// Add metadata about the upload
	if testConfig.Config == nil {
		testConfig.Config = make(map[string]interface{})
	}
	testConfig.Config["_uploadedFile"] = fileHeader.Filename
	testConfig.Config["_uploadedAt"] = time.Now().UTC().Format(time.RFC3339)

	// Register the test
	testDescriptor, err := ah.coordinator.TestRegistry().AddLocalTest(&testConfig)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("failed adding test: %v", err), http.StatusInternalServerError)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), &PostTestsRegisterUploadResponse{
		TestID: testDescriptor.ID(),
		Name:   testDescriptor.Config().Name,
		Config: testDescriptor.Config().Config,
	})
}