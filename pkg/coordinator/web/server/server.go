package server

import (
	"net"
	"net/http"

	coordinator_types "github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/api"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/handlers"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/urfave/negroni"

	// import swagger docs
	_ "github.com/ethpandaops/assertoor/pkg/coordinator/web/api/docs"
)

type WebServer struct {
	serverConfig *types.ServerConfig
	logger       logrus.FieldLogger
	router       *mux.Router
	server       *http.Server
}

func NewWebServer(config *types.ServerConfig, logger logrus.FieldLogger) (*WebServer, error) {
	ws := &WebServer{
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

func (ws *WebServer) StartAPI(config *types.APIConfig, logger logrus.FieldLogger, coordinator coordinator_types.Coordinator) error {
	if !config.Enabled {
		return nil
	}

	ws.router.PathPrefix("/api/docs/").Handler(httpSwagger.WrapHandler)

	// register api routes
	apiHandler := api.NewAPIHandler(logger, coordinator)
	ws.router.HandleFunc("/api/v1/tests", apiHandler.GetTests).Methods("GET")
	ws.router.HandleFunc("/api/v1/test/{testId}", apiHandler.GetTest).Methods("GET")
	ws.router.HandleFunc("/api/v1/test/{testId}/run", apiHandler.PostTestRun).Methods("POST")
	ws.router.HandleFunc("/api/v1/test_runs", apiHandler.GetTestRuns).Methods("GET")
	ws.router.HandleFunc("/api/v1/test_run/{runId}", apiHandler.GetTestRun).Methods("GET")
	ws.router.HandleFunc("/api/v1/test_run/{runId}/details", apiHandler.GetTestRunDetails).Methods("GET")
	ws.router.HandleFunc("/api/v1/test_run/{runId}/status", apiHandler.GetTestRunStatus).Methods("GET")
	//nolint:gocritic // TODO
	// ws.router.HandleFunc("/api/v1/test_run/{runId}/cancel", apiHandler.PostTestRunCancel).Methods("POST")

	return nil
}

func (ws *WebServer) StartFrontend(config *types.FrontendConfig, coordinator coordinator_types.Coordinator) error {
	if config.Pprof {
		// add pprof handler
		ws.router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	}

	if config.Enabled {
		frontend, err := web.NewFrontend(config)
		if err != nil {
			return err
		}

		// register frontend routes
		frontendHandler := handlers.NewFrontendHandler(coordinator)
		ws.router.HandleFunc("/", frontendHandler.Index).Methods("GET")
		ws.router.HandleFunc("/test/{testId}", frontendHandler.TestPage).Methods("GET")
		ws.router.HandleFunc("/run/{runId}", frontendHandler.TestRun).Methods("GET")
		ws.router.HandleFunc("/clients", frontendHandler.Clients).Methods("GET")
		ws.router.HandleFunc("/logs/{since}", frontendHandler.LogsData).Methods("GET")

		ws.router.PathPrefix("/").Handler(frontend)
	}

	return nil
}
