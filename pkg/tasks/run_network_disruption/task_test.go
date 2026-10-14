package runnetworkdisruption

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared fixture keys/names so goconst doesn't trip over repeated literals.
const (
	testKeyName      = "name"
	testKeyTarget    = "target"
	testNameBlackout = "blackout"
	testKeyNodeIndex = "node-index"
	testKeyGroups    = "groups"
)

// fakeDisruptoor is a minimal in-memory stand-in for the disruptoor v1 API:
// it stores whatever state is PUT, serves it back on GET with a naive ETag,
// and honours If-Match / clear semantics.
type fakeDisruptoor struct {
	state    map[string]any
	etag     int
	putCount int
	rejects  string // when non-empty, PUT fails 400 with this error message
}

func (f *fakeDisruptoor) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /v1/state", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", f.currentETag())
		_ = json.NewEncoder(w).Encode(f.state)
	})
	mux.HandleFunc("PUT /v1/state", func(w http.ResponseWriter, r *http.Request) {
		f.putCount++

		if f.rejects != "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": f.rejects})

			return
		}

		if ifMatch := r.Header.Get("If-Match"); ifMatch != "" && ifMatch != f.currentETag() {
			w.WriteHeader(http.StatusPreconditionFailed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "state changed"})

			return
		}

		desired := make(map[string]any, 3)
		if err := json.NewDecoder(r.Body).Decode(&desired); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		f.state = desired
		f.etag++

		_ = json.NewEncoder(w).Encode(f.state)
	})
	mux.HandleFunc("POST /v1/state/clear", func(w http.ResponseWriter, _ *http.Request) {
		f.state = map[string]any{}
		f.etag++

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
	})

	return mux
}

func (f *fakeDisruptoor) currentETag() string {
	return `"` + strconv.Itoa(f.etag) + `"`
}

func newTestTask(t *testing.T, serverURL string, mutate func(c *Config)) *Task {
	t.Helper()

	config := DefaultConfig()
	config.DisruptoorURL = serverURL
	config.AwaitAPITimeout = helper.Duration{Duration: 0}
	mutate(&config)
	require.NoError(t, config.Validate())

	return &Task{
		config:     config,
		logger:     logrus.New(),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func TestPutStateSendsConfiguredEntries(t *testing.T) {
	fake := &fakeDisruptoor{state: map[string]any{}}
	srv := httptest.NewServer(fake.handler())

	defer srv.Close()

	task := newTestTask(t, srv.URL, func(c *Config) {
		c.Isolations = []map[string]any{{
			testKeyName:   testNameBlackout,
			testKeyTarget: map[string]any{testKeyNodeIndex: 1, "client-type": "beacon"},
			"scope":       []any{"cl_p2p", "el_p2p", "include_control"},
		}}
	})

	require.NoError(t, task.putState(context.Background(), task.buildDesiredState(), ""))

	entries := stateEntries(fake.state, stateKeyIsolations)
	require.Len(t, entries, 1)
	assert.Equal(t, testNameBlackout, entries[0][testKeyName])
}

func TestPutStateSurfacesServerError(t *testing.T) {
	fake := &fakeDisruptoor{state: map[string]any{}, rejects: `target cannot be "all"`}
	srv := httptest.NewServer(fake.handler())

	defer srv.Close()

	task := newTestTask(t, srv.URL, func(c *Config) {
		c.Isolations = []map[string]any{{testKeyName: "bad", testKeyTarget: "all"}}
	})

	err := task.putState(context.Background(), task.buildDesiredState(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `target cannot be "all"`)
}

func TestUpdateMergesByNameAndRemoves(t *testing.T) {
	fake := &fakeDisruptoor{state: map[string]any{
		stateKeyPartitions: []any{
			map[string]any{testKeyName: "keep-me", testKeyGroups: []any{"a", "b"}},
		},
		stateKeyIsolations: []any{
			map[string]any{testKeyName: "replace-me", testKeyTarget: map[string]any{testKeyNodeIndex: 1}},
			map[string]any{testKeyName: "remove-me", testKeyTarget: map[string]any{testKeyNodeIndex: 2}},
		},
	}}
	srv := httptest.NewServer(fake.handler())

	defer srv.Close()

	task := newTestTask(t, srv.URL, func(c *Config) {
		c.Action = ActionUpdate
		c.RemoveNames = []string{"remove-me"}
		c.Isolations = []map[string]any{
			{testKeyName: "replace-me", testKeyTarget: map[string]any{testKeyNodeIndex: 3}},
			{testKeyName: "brand-new", testKeyTarget: map[string]any{testKeyNodeIndex: 4}},
		}
	})

	require.NoError(t, task.updateState(context.Background()))

	partitions := stateEntries(fake.state, stateKeyPartitions)
	require.Len(t, partitions, 1)
	assert.Equal(t, "keep-me", partitions[0][testKeyName])

	isolations := stateEntries(fake.state, stateKeyIsolations)
	require.Len(t, isolations, 2)
	assert.Equal(t, "replace-me", isolations[0][testKeyName])
	assert.Equal(t, map[string]any{testKeyNodeIndex: float64(3)}, isolations[0][testKeyTarget])
	assert.Equal(t, "brand-new", isolations[1][testKeyName])
}

func TestClearStateHealsEverything(t *testing.T) {
	fake := &fakeDisruptoor{state: map[string]any{
		stateKeyIsolations: []any{map[string]any{testKeyName: testNameBlackout}},
	}}
	srv := httptest.NewServer(fake.handler())

	defer srv.Close()

	task := newTestTask(t, srv.URL, func(c *Config) {
		c.Action = ActionClear
	})

	require.NoError(t, task.clearState(context.Background()))
	assert.Empty(t, fake.state)
}

func TestUpdateRetriesOnStaleETag(t *testing.T) {
	fake := &fakeDisruptoor{state: map[string]any{}}
	// Bump the ETag underneath the first PUT to simulate a concurrent writer.
	raced := false
	mux := http.NewServeMux()
	mux.Handle("/", fake.handler())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && !raced {
			raced = true
			f := fake
			f.etag++
		}

		mux.ServeHTTP(w, r)
	}))

	defer srv.Close()

	task := newTestTask(t, srv.URL, func(c *Config) {
		c.Action = ActionUpdate
		c.Isolations = []map[string]any{{testKeyName: testNameBlackout, testKeyTarget: map[string]any{testKeyNodeIndex: 1}}}
	})

	require.NoError(t, task.updateState(context.Background()))
	require.Equal(t, 2, fake.putCount, "first PUT should 412, second should succeed")

	isolations := stateEntries(fake.state, stateKeyIsolations)
	require.Len(t, isolations, 1)
}

func TestProbeHealth(t *testing.T) {
	fake := &fakeDisruptoor{state: map[string]any{}}
	srv := httptest.NewServer(fake.handler())

	defer srv.Close()

	task := newTestTask(t, srv.URL, func(c *Config) {
		c.Action = ActionClear
	})

	require.NoError(t, task.probeHealth(context.Background()))

	srv.Close()
	require.Error(t, task.probeHealth(context.Background()))
}
