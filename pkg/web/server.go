package web

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/ethpandaops/assertoor/pkg/events"
	coordinator_types "github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/web/api"
	"github.com/ethpandaops/assertoor/pkg/web/auth"
	"github.com/ethpandaops/assertoor/pkg/web/handlers"
	"github.com/ethpandaops/assertoor/pkg/web/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/urfave/negroni"

	// import swagger docs
	_ "github.com/ethpandaops/assertoor/pkg/web/api/docs"

	// import pprof
	//nolint:gosec // ignore
	_ "net/http/pprof"
)

type Server struct {
	serverConfig *types.ServerConfig
	logger       logrus.FieldLogger
	router       *mux.Router
	server       *http.Server
}

func NewWebServer(config *types.ServerConfig, logger logrus.FieldLogger) (*Server, error) {
	ws := &Server{
		serverConfig: config,
		logger:       logger.WithField("module", "web"),
		router:       mux.NewRouter(),
	}

	n := negroni.New()
	n.Use(negroni.NewRecovery())
	n.UseHandler(ws.router)

	if config.Host == "" {
		config.Host = "0.0.0.0"
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	ws.server = &http.Server{
		Addr:         config.Host + ":" + config.Port,
		WriteTimeout: config.WriteTimeout,
		ReadTimeout:  config.ReadTimeout,
		IdleTimeout:  config.IdleTimeout,
		Handler:      n,
	}

	listener, err := net.Listen("tcp", config.Host+":"+config.Port)
	if err != nil {
		return nil, err
	}

	go func() {
		err := ws.server.Serve(listener)
		if err != nil {
			ws.logger.Errorf("HTTP server serve error: %v", err)
		}
	}()

	return ws, nil
}

func (ws *Server) ConfigureRoutes(webConfig *types.WebConfig, aiConfig *types.AIConfig, coordinator coordinator_types.Coordinator, eventBus *events.EventBus) error {
	frontendConfig := webConfig.Frontend
	apiConfig := webConfig.API
	isAPIEnabled := apiConfig != nil && apiConfig.Enabled
	isFrontendEnabled := frontendConfig != nil && frontendConfig.Enabled

	// Build auth handler. When AuthProviderURL is empty the handler runs in
	// open mode and every request is authorized. When set, incoming bearer
	// tokens are verified against the remote authenticatoor's JWKS.
	authHandler, err := auth.NewAuthHandler(context.Background(), ws.logger.WithField("module", "auth"), webConfig.AuthProviderURL, webConfig.AuthProviderAudience)
	if err != nil {
		return err
	}

	if (isAPIEnabled || isFrontendEnabled) && authHandler.IsOpen() {
		ws.logger.Warn("authProviderUrl is empty: API endpoints are unauthenticated. Configure an authenticatoor URL to require auth.")
	}

	if isAPIEnabled {
		// register api routes
		apiHandler := api.NewAPIHandler(ws.logger.WithField("module", "api"), coordinator, authHandler)

		// public apis
		ws.router.HandleFunc("/api/v1/tests", apiHandler.GetTests).Methods("GET")
		ws.router.HandleFunc("/api/v1/test/{testId}", apiHandler.GetTest).Methods("GET")
		ws.router.HandleFunc("/api/v1/test/{testId}/yaml", apiHandler.GetTestYaml).Methods("GET")
		ws.router.HandleFunc("/api/v1/test/{testId}/latest_result", apiHandler.GetTestLatestResult).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_runs", apiHandler.GetTestRuns).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}", apiHandler.GetTestRun).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/result", apiHandler.GetTestRunResult).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/status", apiHandler.GetTestRunStatus).Methods("GET")
		ws.router.HandleFunc("/api/v1/task_descriptors", apiHandler.GetTaskDescriptors).Methods("GET")
		ws.router.HandleFunc("/api/v1/task_descriptor/{name}", apiHandler.GetTaskDescriptor).Methods("GET")
		ws.router.HandleFunc("/api/v1/clients", apiHandler.GetClients).Methods("GET")
		ws.router.HandleFunc("/api/v1/global_variables", apiHandler.GetGlobalVariables).Methods("GET")
		ws.router.HandleFunc("/api/v1/version", apiHandler.GetVersion).Methods("GET")
		ws.router.HandleFunc("/api/v1/playbook_library", apiHandler.GetPlaybookLibrary).Methods("GET")
		ws.router.HandleFunc("/api/v1/playbook_library/check", apiHandler.GetPlaybookLibraryCheck).Methods("GET")
		ws.router.HandleFunc("/api/v1/network_status", apiHandler.GetNetworkStatus).Methods("GET")
		ws.router.HandleFunc("/api/v1/dashboard_config", apiHandler.GetDashboardConfig).Methods("GET")
		ws.router.HandleFunc("/api/v1/dashboard_config", apiHandler.PutDashboardConfig).Methods("PUT")

		// SSE event stream endpoints
		if eventBus != nil {
			sseHandler := events.NewSSEHandlerWithAuth(
				ws.logger.WithField("module", "sse"),
				eventBus,
				authHandler.CheckAuthToken,
				!authHandler.IsOpen(), // Require auth for log events when not in open mode
			)

			ws.router.HandleFunc("/api/v1/events/stream", sseHandler.HandleGlobalStream).Methods("GET")
			ws.router.HandleFunc("/api/v1/events/clients", sseHandler.HandleClientStream).Methods("GET")
			ws.router.HandleFunc("/api/v1/test_run/{runId}/events", func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)

				runID, err := strconv.ParseUint(vars["runId"], 10, 64)
				if err != nil {
					http.Error(w, "Invalid run ID", http.StatusBadRequest)
					return
				}

				sseHandler.HandleTestRunStream(w, r, runID)
			}).Methods("GET")
		}

		// Logs API (protected)
		ws.router.HandleFunc("/api/v1/logs/{since}", apiHandler.GetLogs).Methods("GET")
		ws.router.HandleFunc("/logs/{since}", apiHandler.GetLogs).Methods("GET") // Legacy alias for external tools

		// protected apis (require authentication)
		ws.router.HandleFunc("/api/v1/tests/register", apiHandler.PostTestsRegister).Methods("POST")
		ws.router.HandleFunc("/api/v1/tests/register_external", apiHandler.PostTestsRegisterExternal).Methods("POST")
		ws.router.HandleFunc("/api/v1/tests/delete", apiHandler.PostTestsDelete).Methods("POST")
		ws.router.HandleFunc("/api/v1/test_run", apiHandler.PostTestRunsSchedule).Methods("POST") // legacy
		ws.router.HandleFunc("/api/v1/test_runs/schedule", apiHandler.PostTestRunsSchedule).Methods("POST")
		ws.router.HandleFunc("/api/v1/test_runs/delete", apiHandler.PostTestRunsDelete).Methods("POST")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/cancel", apiHandler.PostTestRunCancel).Methods("POST")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/details", apiHandler.GetTestRunDetails).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/task/{taskIndex}/details", apiHandler.GetTestRunTaskDetails).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/task/{taskId}/result/{resultType}/{fileId:.*}", apiHandler.GetTaskResult).Methods("GET")

		// AI endpoints (if enabled)
		if aiConfig != nil && aiConfig.Enabled {
			aiHandler := api.NewAIHandler(
				aiConfig,
				coordinator.Database(),
				ws.logger.WithField("module", "ai-api"),
				authHandler,
			)
			ws.router.HandleFunc("/api/v1/ai/config", aiHandler.GetConfig).Methods("GET")
			ws.router.HandleFunc("/api/v1/ai/system_prompt", aiHandler.GetSystemPrompt).Methods("GET")
			ws.router.HandleFunc("/api/v1/ai/validate", aiHandler.ValidateYaml).Methods("POST")
			ws.router.HandleFunc("/api/v1/ai/usage", aiHandler.GetUsage).Methods("GET")
			ws.router.HandleFunc("/api/v1/ai/chat", aiHandler.Chat).Methods("POST")
			ws.router.HandleFunc("/api/v1/ai/chat/{sessionId}", aiHandler.GetChatSession).Methods("GET")
			ws.router.HandleFunc("/api/v1/ai/chat/{sessionId}/stream", aiHandler.StreamChatSession).Methods("GET")
		}

		// Swagger API documentation (standalone, no custom header)
		ws.router.PathPrefix("/api/docs/").Handler(httpSwagger.Handler(func(c *httpSwagger.Config) {
			c.Layout = httpSwagger.StandaloneLayout
		}))
	}

	if frontendConfig != nil {
		if frontendConfig.Pprof {
			// add pprof handler
			ws.router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
		}

		if frontendConfig.Enabled {
			// Create SPA handler for React frontend, injecting the runtime
			// config so the SPA can wire itself to the same authenticatoor.
			spaHandler, err := handlers.NewSPAHandler(
				ws.logger.WithField("module", "web-spa"),
				handlers.RuntimeConfig{
					AuthProviderURL: webConfig.AuthProviderURL,
				},
			)
			if err != nil {
				return err
			}

			// SPA handles all frontend routes
			ws.router.PathPrefix("/").Handler(spaHandler)
		}
	}

	return nil
}
