package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/static"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/templates"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/utils"
	"github.com/sirupsen/logrus"
)

var LayoutTemplateFiles = []string{
	"_layout/layout.html",
	"_layout/header.html",
	"_layout/footer.html",
}

type FrontendHandler struct {
	coordinator     types.Coordinator
	logger          logrus.FieldLogger
	templates       *templates.Templates
	defaultHandler  http.Handler
	rootFileSys     http.FileSystem
	siteName        string
	minifyHTML      bool
	debugMode       bool
	securityTrimmed bool
	isAPIEnabled    bool
}

func NewFrontendHandler(coordinator types.Coordinator, logger logrus.FieldLogger, siteName string, minifyHTML, debugMode, securityTrimmed, apiEnabled bool) *FrontendHandler {
	fileSys := http.FS(static.FS)

	if siteName == "" {
		siteName = "Assertoor"
	}

	return &FrontendHandler{
		coordinator:     coordinator,
		logger:          logger,
		templates:       templates.New(logger, utils.GetTemplateFuncs(), minifyHTML),
		defaultHandler:  http.FileServer(fileSys),
		rootFileSys:     fileSys,
		siteName:        siteName,
		minifyHTML:      minifyHTML,
		debugMode:       debugMode,
		securityTrimmed: securityTrimmed,
		isAPIEnabled:    apiEnabled,
	}
}

type PageData struct {
	Active          string
	Meta            *Meta
	ShowSidebar     bool
	SidebarData     interface{}
	Data            interface{}
	Version         string
	Year            int
	Title           string
	Lang            string
	Debug           bool
	SecurityTrimmed bool
	IsAPIEnabled    bool
	DebugTemplates  []string
}

type Meta struct {
	Title       string
	Description string
	Domain      string
	Path        string
	Tlabel1     string
	Tdata1      string
	Tlabel2     string
	Tdata2      string
	Templates   string
}

type ErrorPageData struct {
	CallTime   time.Time
	CallURL    string
	ErrorMsg   string
	StackTrace string
	Version    string
}

func (fh *FrontendHandler) initPageData(r *http.Request, active, pagePath, pageTitle string, mainTemplates []string) *PageData {
	fullTitle := fmt.Sprintf("%v - %v - %v", pageTitle, fh.siteName, time.Now().Year())

	if pageTitle == "" {
		fullTitle = fmt.Sprintf("%v - %v", fh.siteName, time.Now().Year())
	}

	host := ""
	if r != nil {
		host = r.Host
	}

	data := &PageData{
		Meta: &Meta{
			Title:       fullTitle,
			Description: "assertoor: testnet testing tool",
			Domain:      host,
			Path:        pagePath,
			Templates:   strings.Join(mainTemplates, ","),
		},
		Active:          active,
		Data:            &struct{}{},
		Version:         buildinfo.GetVersion(),
		Year:            time.Now().UTC().Year(),
		Title:           fh.siteName,
		Lang:            "en-US",
		Debug:           fh.debugMode,
		SecurityTrimmed: fh.securityTrimmed,
		IsAPIEnabled:    fh.isAPIEnabled,
	}

	if r != nil {
		acceptedLangs := strings.Split(r.Header.Get("Accept-Language"), ",")
		if len(acceptedLangs) > 0 {
			if strings.Contains(acceptedLangs[0], "ru") || strings.Contains(acceptedLangs[0], "RU") {
				data.Lang = "ru-RU"
			}
		}

		for _, v := range r.Cookies() {
			if v.Name == "language" {
				data.Lang = v.Value
				break
			}
		}
	}

	return data
}

func (fh *FrontendHandler) handleTemplateError(w http.ResponseWriter, r *http.Request, fileIdentifier, functionIdentifier string, err error) error {
	// ignore network related errors
	if err != nil && !errors.Is(err, syscall.EPIPE) && !errors.Is(err, syscall.ETIMEDOUT) {
		fh.logger.WithFields(logrus.Fields{
			"file":       fileIdentifier,
			"function":   functionIdentifier,
			"error type": fmt.Sprintf("%T", err),
			"route":      r.URL.String(),
		}).WithError(err).Error("error executing template")

		http.Error(w, "Internal server error", http.StatusServiceUnavailable)

		return err
	}

	return err
}

func (fh *FrontendHandler) HandlePageError(w http.ResponseWriter, r *http.Request, pageError error) {
	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles, "_layout/500.html")
	notFoundTemplate := fh.templates.GetTemplate(templateFiles...)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusInternalServerError)

	data := fh.initPageData(r, "blockchain", r.URL.Path, "Internal Error", templateFiles)
	errData := &ErrorPageData{
		CallTime: time.Now(),
		CallURL:  r.URL.String(),
		ErrorMsg: pageError.Error(),
		Version:  buildinfo.GetVersion(),
	}
	data.Data = errData

	err := notFoundTemplate.ExecuteTemplate(w, "layout", data)
	if err != nil {
		logrus.Errorf("error executing page error template for %v route: %v", r.URL.String(), err)
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)

		return
	}
}

func (fh *FrontendHandler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles, "_layout/404.html")
	notFoundTemplate := fh.templates.GetTemplate(templateFiles...)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)

	data := fh.initPageData(r, "blockchain", r.URL.Path, "Not Found", templateFiles)

	err := notFoundTemplate.ExecuteTemplate(w, "layout", data)
	if err != nil {
		logrus.Errorf("error executing not-found template for %v route: %v", r.URL.String(), err)
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)

		return
	}
}

func (fh *FrontendHandler) BuildPageHeader() (string, error) {
	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles, "_layout/blank.html")
	blankTemplate := fh.templates.GetTemplate(templateFiles...)

	data := fh.initPageData(nil, "blank", "", "", templateFiles)

	var outBuf bytes.Buffer

	err := blankTemplate.ExecuteTemplate(&outBuf, "header", data)
	if err != nil {
		return "", fmt.Errorf("error executing blank template: %v", err)
	}

	return outBuf.String(), nil
}

func (fh *FrontendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// basically a copy of http.FileServer and of the first lines http.serveFile functions
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	name := path.Clean(upath)

	f, err := fh.rootFileSys.Open(name)
	if err != nil {
		fh.handleHTTPError(err, fh.HandleNotFound, w, r)
		return
	}

	defer func() {
		if err2 := f.Close(); err2 != nil {
			logrus.WithError(err2).Warn("failed to close file")
		}
	}()

	_, err = f.Stat()
	if err != nil {
		fh.handleHTTPError(err, fh.HandleNotFound, w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/") {
		fh.handleHTTPError(fs.ErrNotExist, fh.HandleNotFound, w, r)
		return
	}

	fh.defaultHandler.ServeHTTP(w, r)
}

func (fh *FrontendHandler) handleHTTPError(err error, handler func(http.ResponseWriter, *http.Request), w http.ResponseWriter, r *http.Request) {
	// If error is 404, use custom handler
	if errors.Is(err, fs.ErrNotExist) {
		handler(w, r)
		return
	}
	// otherwise serve http error
	if errors.Is(err, fs.ErrPermission) {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
	// Default:
	logrus.WithError(err).Errorf("page handler error")
	http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
}
