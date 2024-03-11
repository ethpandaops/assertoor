package server

import (
	"encoding/json"
	"html/template"
	"net"
	"net/http"
	"strings"

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

func (ws *WebServer) ConfigureRoutes(config *types.WebConfig, logger logrus.FieldLogger, coordinator coordinator_types.Coordinator) error {
	var frontend *web.Frontend

	if config.Frontend != nil && config.Frontend.Enabled {
		var err error

		frontend, err = web.NewFrontend(config.Frontend)
		if err != nil {
			return err
		}
	}

	if config.API != nil && config.API.Enabled {
		ws.router.PathPrefix("/api/docs/").Handler(ws.getSwaggerHandler(logger))

		// register api routes
		apiHandler := api.NewAPIHandler(logger.WithField("module", "api"), coordinator)
		ws.router.HandleFunc("/api/v1/tests", apiHandler.GetTests).Methods("GET")
		ws.router.HandleFunc("/api/v1/tests/register", apiHandler.PostTestsRegister).Methods("POST")
		ws.router.HandleFunc("/api/v1/tests/register_external", apiHandler.PostTestsRegisterExternal).Methods("POST")
		ws.router.HandleFunc("/api/v1/test/{testId}", apiHandler.GetTest).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_runs", apiHandler.GetTestRuns).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run", apiHandler.PostTestRunsSchedule).Methods("POST") // legacy
		ws.router.HandleFunc("/api/v1/test_runs/schedule", apiHandler.PostTestRunsSchedule).Methods("POST")
		ws.router.HandleFunc("/api/v1/test_run/{runId}", apiHandler.GetTestRun).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/details", apiHandler.GetTestRunDetails).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/status", apiHandler.GetTestRunStatus).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/cancel", apiHandler.PostTestRunCancel).Methods("POST")
	}

	if config.Frontend != nil {
		// if config.Frontend.Pprof {
		// add pprof handler
		ws.router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
		// }

		if config.Frontend.Enabled {
			frontendHandler := handlers.NewFrontendHandler(coordinator)

			ws.router.HandleFunc("/", frontendHandler.Index).Methods("GET")
			ws.router.HandleFunc("/test/{testId}", frontendHandler.TestPage).Methods("GET")
			ws.router.HandleFunc("/run/{runId}", frontendHandler.TestRun).Methods("GET")
			ws.router.HandleFunc("/clients", frontendHandler.Clients).Methods("GET")
			ws.router.HandleFunc("/logs/{since}", frontendHandler.LogsData).Methods("GET")

			ws.router.PathPrefix("/").Handler(frontend)
		}
	}

	return nil
}

func (ws *WebServer) getSwaggerHandler(logger logrus.FieldLogger) http.HandlerFunc {
	return httpSwagger.Handler(func(c *httpSwagger.Config) {
		c.Layout = httpSwagger.StandaloneLayout

		// override swagger header bar
		headerHTML, err := web.BuildPageHeader()
		if err != nil {
			logger.Errorf("failed generating page header for api: %v", err)
		} else {
			headerStr, err := json.Marshal(headerHTML)
			if err != nil {
				logger.Errorf("failed marshalling page header for api: %v", err)
			} else {
				var headerScript strings.Builder

				headerScript.WriteString("var headerHtml = ")
				headerScript.Write(headerStr)
				headerScript.WriteString(";")
				headerScript.WriteString("var headerEl = document.createElement(\"div\"); headerEl.className = \"header\"; headerEl.innerHTML = headerHtml; document.body.insertBefore(headerEl, document.body.firstElementChild);")
				headerScript.WriteString(`function addCss(fileName) { var el = document.createElement("link"); el.type = "text/css"; el.rel = "stylesheet"; el.href = fileName; document.head.appendChild(el); }`)
				headerScript.WriteString(`function addStyle(cssCode) { var el = document.createElement("style"); el.type = "text/css"; el.appendChild(document.createTextNode(cssCode)); document.head.appendChild(el); }`)
				headerScript.WriteString(`function addScript(fileName) { var el = document.createElement("script"); el.type = "text/javascript"; el.src = fileName; document.head.appendChild(el); }`)
				headerScript.WriteString(`addCss("/css/bootstrap.min.css");`)
				headerScript.WriteString(`addCss("/css/layout.css");`)
				headerScript.WriteString(`addScript("/js/color-modes.js");`)
				headerScript.WriteString(`addScript("/js/jquery.min.js");`)
				headerScript.WriteString(`addScript("/js/bootstrap.bundle.min.js");`)
				headerScript.WriteString(`addStyle("#swagger-ui .topbar { display: none; } .swagger-ui .opblock .opblock-section-header { background: rgba(var(--bs-body-bg-rgb), 0.8); } [data-bs-theme='dark'] .swagger-ui svg { filter: invert(100%); }");`)
				headerScript.WriteString(`
					// override swagger style (replace all color selectors)
					swaggerStyle = Array.prototype.filter.call(document.styleSheets, function(style) { return style.href && style.href.match(/swagger-ui/) })[0];
					swaggerRules = swaggerStyle.rules || swaggerStyle.cssRules;
					swaggerColorSelectors = [];
					Array.prototype.forEach.call(swaggerRules, function(rule) {
						if(rule.cssText.match(/color: rgb\(59, 65, 81\);/)) {
							swaggerColorSelectors.push(rule.selectorText);
						}
					});
					addStyle(swaggerColorSelectors.join(", ") + " { color: inherit; }");

				`)

				//nolint:gosec // ignore
				c.AfterScript = template.JS(headerScript.String())
			}
		}
	})
}
