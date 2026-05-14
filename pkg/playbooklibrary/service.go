package playbooklibrary

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// httpClientTimeout caps how long a single remote fetch may take.
const httpClientTimeout = 30 * time.Second

type service struct {
	cfg      *Config
	log      logrus.FieldLogger
	provider LocalTestProvider
	client   *http.Client

	// baseURL is resolved once from cfg.BaseURL || derive(cfg.IndexURL).
	baseURL string

	mu       sync.RWMutex
	cache    *Index
	cachedAt time.Time
	cacheTTL time.Duration
}

// NewService constructs a Service. Returns nil when cfg is nil or
// cfg.Enabled is false so callers can use the nil-check as their
// "feature disabled" gate.
func NewService(cfg *Config, log logrus.FieldLogger, provider LocalTestProvider) Service {
	if cfg == nil || !cfg.Enabled {
		return &service{cfg: cfg, log: log, provider: provider}
	}

	indexURL := cfg.IndexURL
	if indexURL == "" {
		indexURL = DefaultIndexURL
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = deriveBaseURL(indexURL)
	}

	ttl := cfg.CacheTTL.Duration
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	return &service{
		cfg:      cfg,
		log:      log.WithField("component", "playbook-library"),
		provider: provider,
		client:   &http.Client{Timeout: httpClientTimeout},
		baseURL:  baseURL,
		cacheTTL: ttl,
	}
}

func (s *service) Enabled() bool {
	return s.cfg != nil && s.cfg.Enabled
}

func (s *service) GetIndex(ctx context.Context) (*IndexResponse, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("playbook library is disabled")
	}

	idx, err := s.cachedIndex(ctx)
	if err != nil {
		return nil, err
	}

	return &IndexResponse{
		Generated: idx.Generated,
		BaseURL:   s.baseURL,
		IndexURL:  s.indexURL(),
		Folders:   idx.Folders,
		Playbooks: idx.Playbooks,
	}, nil
}

func (s *service) Check(ctx context.Context, file string) (*CheckResult, string, error) {
	if !s.Enabled() {
		return nil, "", fmt.Errorf("playbook library is disabled")
	}

	if file == "" {
		return nil, "", fmt.Errorf("file parameter is required")
	}

	// Validate against the index so callers can't drive arbitrary
	// requests through this endpoint.
	idx, err := s.cachedIndex(ctx)
	if err != nil {
		return nil, "", err
	}

	var entry *PlaybookEntry

	for i := range idx.Playbooks {
		if idx.Playbooks[i].File == file {
			entry = &idx.Playbooks[i]
			break
		}
	}

	if entry == nil {
		return nil, "", fmt.Errorf("playbook %q not found in index", file)
	}

	remoteURL := s.baseURL + entry.File

	remoteYaml, err := s.fetch(ctx, remoteURL)
	if err != nil {
		return nil, "", fmt.Errorf("fetch remote playbook: %w", err)
	}

	result := &CheckResult{
		State:      CheckStateAbsent,
		RemoteID:   entry.ID,
		RemoteName: entry.Name,
		RemoteURL:  remoteURL,
	}

	if s.provider == nil {
		return result, remoteYaml, nil
	}

	localYaml, localName, err := s.provider.FindLocalYaml(ctx, entry.ID)
	if err != nil {
		return nil, "", fmt.Errorf("look up local test: %w", err)
	}

	if localYaml == "" {
		return result, remoteYaml, nil
	}

	result.LocalTestID = entry.ID
	result.LocalName = localName
	result.LocalSource = localYaml

	if normalizeYaml(localYaml) == normalizeYaml(remoteYaml) {
		result.State = CheckStateSame
	} else {
		result.State = CheckStateDifferent
	}

	return result, remoteYaml, nil
}

// cachedIndex returns the cached index, fetching a fresh copy when the
// TTL has elapsed or no cache exists. A stale cache is returned with a
// warning if the refresh fails.
func (s *service) cachedIndex(ctx context.Context) (*Index, error) {
	s.mu.RLock()
	fresh := s.cache != nil && time.Since(s.cachedAt) < s.cacheTTL
	cached := s.cache
	s.mu.RUnlock()

	if fresh {
		return cached, nil
	}

	idx, err := s.fetchIndex(ctx)
	if err != nil {
		if cached != nil {
			s.log.WithError(err).Warn("failed to refresh playbook library index, serving stale copy")
			return cached, nil
		}

		return nil, err
	}

	s.mu.Lock()
	s.cache = idx
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return idx, nil
}

func (s *service) fetchIndex(ctx context.Context) (*Index, error) {
	body, err := s.fetch(ctx, s.indexURL())
	if err != nil {
		return nil, err
	}

	idx := &Index{}
	if err := yaml.Unmarshal([]byte(body), idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}

	return idx, nil
}

func (s *service) fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.log.WithError(closeErr).Warn("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: HTTP %d %s", url, resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}

	return string(body), nil
}

func (s *service) indexURL() string {
	if s.cfg != nil && s.cfg.IndexURL != "" {
		return s.cfg.IndexURL
	}

	return DefaultIndexURL
}

// deriveBaseURL strips the last path segment from an index URL so it
// can be used as a prefix for `file` entries.
func deriveBaseURL(indexURL string) string {
	idx := strings.LastIndex(indexURL, "/")
	if idx < 0 {
		return indexURL
	}

	return indexURL[:idx+1]
}

// normalizeYaml trims trailing whitespace from each line and the
// surrounding document so equivalent files don't get flagged as
// different just because of editor whitespace drift.
func normalizeYaml(s string) string {
	trimmed := strings.TrimSpace(s)
	lines := strings.Split(trimmed, "\n")

	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}

	return strings.Join(lines, "\n")
}
