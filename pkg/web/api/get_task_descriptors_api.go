package api

import (
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/tasks"
	"github.com/gorilla/mux"
)

// GetTaskDescriptors godoc
// @Summary Get all task descriptors
// @Description Returns a list of all available task descriptors with their JSON schemas
// @Tags Task
// @Produce json
// @Success 200 {object} Response{data=[]tasks.TaskDescriptorAPI}
// @Router /api/v1/task_descriptors [get]
func (ah *APIHandler) GetTaskDescriptors(w http.ResponseWriter, r *http.Request) {
	descriptors := tasks.GetAllTaskDescriptorsAPI()
	ah.sendOKResponse(w, r.URL.String(), descriptors)
}

// GetTaskDescriptor godoc
// @Summary Get a specific task descriptor
// @Description Returns a single task descriptor by name with its JSON schema
// @Tags Task
// @Produce json
// @Param name path string true "Task name"
// @Success 200 {object} Response{data=tasks.TaskDescriptorAPI}
// @Failure 404 {object} Response
// @Router /api/v1/task_descriptor/{name} [get]
func (ah *APIHandler) GetTaskDescriptor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	descriptor := tasks.GetTaskDescriptorAPI(name)
	if descriptor == nil {
		ah.sendErrorResponse(w, r.URL.String(), "task not found", http.StatusNotFound)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), descriptor)
}
