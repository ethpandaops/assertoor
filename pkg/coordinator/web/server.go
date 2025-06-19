package web

import (
	"encoding/json"
	"html/template"
	"net"
	"net/http"
	"strings"

	coordinator_types "github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/web/api"
	"github.com/erigontech/assertoor/pkg/coordinator/web/handlers"
	"github.com/erigontech/assertoor/pkg/coordinator/web/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/urfave/negroni"

	// import swagger docs
	_ "github.com/erigontech/assertoor/pkg/coordinator/web/api/docs"

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

func (ws *Server) ConfigureRoutes(frontendConfig *types.FrontendConfig, apiConfig *types.APIConfig, coordinator coordinator_types.Coordinator, securityTrimmed bool) error {
	isAPIEnabled := apiConfig != nil && apiConfig.Enabled
	if isAPIEnabled {
		// register api routes
		apiHandler := api.NewAPIHandler(ws.logger.WithField("module", "api"), coordinator)

		// public apis
		ws.router.HandleFunc("/api/v1/tests", apiHandler.GetTests).Methods("GET")
		ws.router.HandleFunc("/api/v1/test/{testId}", apiHandler.GetTest).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_runs", apiHandler.GetTestRuns).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}", apiHandler.GetTestRun).Methods("GET")
		ws.router.HandleFunc("/api/v1/test_run/{runId}/status", apiHandler.GetTestRunStatus).Methods("GET")

		// private apis
		if !securityTrimmed {
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
		}
	}

	if frontendConfig != nil {
		if /*frontendConfig.Pprof &&*/ !securityTrimmed {
			// add pprof handler
			ws.router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
		}

		if frontendConfig.Enabled {
			frontendHandler := handlers.NewFrontendHandler(
				coordinator,
				ws.logger.WithField("module", "web-frontend"),
				frontendConfig.SiteName,
				frontendConfig.Minify,
				frontendConfig.Debug,
				securityTrimmed,
				isAPIEnabled,
			)

			ws.router.HandleFunc("/", frontendHandler.Index).Methods("GET")
			ws.router.HandleFunc("/registry", frontendHandler.Registry).Methods("GET")
			ws.router.HandleFunc("/test/{testId}", frontendHandler.TestPage).Methods("GET")
			ws.router.HandleFunc("/run/{runId}", frontendHandler.TestRun).Methods("GET")
			ws.router.HandleFunc("/clients", frontendHandler.Clients).Methods("GET")
			ws.router.HandleFunc("/logs/{since}", frontendHandler.LogsData).Methods("GET")

			if isAPIEnabled {
				// add swagger handler
				ws.router.PathPrefix("/api/docs/").Handler(ws.getSwaggerHandler(ws.logger, frontendHandler))
			}

			ws.router.PathPrefix("/").Handler(frontendHandler)
		}
	}

	return nil
}

func (ws *Server) getSwaggerHandler(logger logrus.FieldLogger, fh *handlers.FrontendHandler) http.HandlerFunc {
	return httpSwagger.Handler(func(c *httpSwagger.Config) {
		c.Layout = httpSwagger.StandaloneLayout

		// override swagger header bar
		headerHTML, err := fh.BuildPageHeader()
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
