package api

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
)

// GetTaskResult godoc
// @Id getTaskResult
// @Summary Get task result file
// @Tags TestRun
// @Description Returns a specific result file from a task
// @Produce octet-stream
// @Param runId path string true "ID of the test run"
// @Param taskId path string true "ID of the task"
// @Param resultType path string true "Type of file to return (summary, result, ...)"
// @Param fileId path string true "Index or name of the result file"
// @Success 200 {file} binary "Success"
// @Failure 400 {object} Response "Bad Request"
// @Failure 404 {object} Response "Not Found"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_run/{runId}/task/{taskIndex}/result/{resultType}/{fileId} [get]
func (ah *APIHandler) GetTaskResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	runID, err := strconv.ParseUint(vars["runId"], 10, 64)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "invalid runId provided", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.ParseUint(vars["taskId"], 10, 64)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "invalid taskId provided", http.StatusBadRequest)
		return
	}

	resultType := vars["resultType"]
	if resultType == "" {
		ah.sendErrorResponse(w, r.URL.String(), "invalid resultType provided", http.StatusBadRequest)
		return
	}

	fileID := vars["fileId"]
	if fileID == "" {
		ah.sendErrorResponse(w, r.URL.String(), "invalid fileId provided", http.StatusBadRequest)
		return
	}

	testInstance := ah.coordinator.GetTestByRunID(runID)
	if testInstance == nil {
		ah.sendErrorResponse(w, r.URL.String(), "test run not found", http.StatusNotFound)
		return
	}

	taskScheduler := testInstance.GetTaskScheduler()
	if taskScheduler == nil {
		ah.sendErrorResponse(w, r.URL.String(), "task scheduler not found", http.StatusNotFound)
		return
	}

	// Find task by index
	taskState := taskScheduler.GetTaskState(types.TaskIndex(taskID))
	if taskState == nil {
		ah.sendErrorResponse(w, r.URL.String(), "task not found", http.StatusNotFound)
		return
	}

	// Find the requested result file
	var resultFile *db.TaskResult

	if fileIndex, err := strconv.ParseInt(fileID, 10, 32); err == nil {
		// Search by index
		resultFile, _ = ah.coordinator.Database().GetTaskResultByIndex(runID, uint64(taskState.Index()), resultType, int(fileIndex))
	} else {
		// Search by name
		resultFile, _ = ah.coordinator.Database().GetTaskResultByName(runID, uint64(taskState.Index()), resultType, fileID)
	}

	if resultFile == nil {
		ah.sendErrorResponse(w, r.URL.String(), "result file not found", http.StatusNotFound)
		return
	}

	// Check if view parameter is set
	downloadMode := r.URL.Query().Has("download")

	// Determine content type
	contentType := "application/octet-stream"

	if !downloadMode {
		ext := strings.ToLower(filepath.Ext(resultFile.Name))
		switch ext {
		case ".txt", ".log", ".yaml", ".yml", ".json", ".md":
			contentType = "text/plain"
		case ".html", ".htm":
			contentType = "text/html"
		case ".css":
			contentType = "text/css"
		case ".png":
			contentType = "image/png"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".pdf":
			contentType = "application/pdf"
		default:
			// Default to text/plain for unknown types in view mode
			contentType = "text/plain"
		}
	}

	w.Header().Set("Content-Type", contentType)

	if downloadMode {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", resultFile.Name))
	}

	http.ServeContent(w, r, resultFile.Name, time.Now(), bytes.NewReader(resultFile.Data))
}
