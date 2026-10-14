package vars_test

import (
	"testing"

	"github.com/ethpandaops/assertoor/pkg/vars"
)

// TestResolveQuerySurfacesRuntimeError verifies that a gojq runtime error is returned
// as an error rather than as a successful value. gojq yields evaluation errors as
// values of type error with ok=true, so without an explicit check an erroring query
// (here, adding a number to a string) was reported as a result. In the task `if:`
// path that meant the condition was silently treated as non-boolean and the task was
// skipped instead of failing.
func TestResolveQuerySurfacesRuntimeError(t *testing.T) {
	scope := vars.NewVariables(nil)
	scope.SetVar("n", "hello")

	val, ok, err := scope.ResolveQuery(".n + 1")
	if err == nil {
		t.Fatalf("expected an error for an erroring query, got value %v (ok=%v)", val, ok)
	}

	if _, isErr := val.(error); isErr {
		t.Fatal("the gojq error value leaked out as the result value")
	}
}

// TestResolveQueryValidQuery guards the happy path so the error check does not reject
// good queries.
func TestResolveQueryValidQuery(t *testing.T) {
	scope := vars.NewVariables(nil)
	scope.SetVar("greeting", "hello")

	val, ok, err := scope.ResolveQuery(".greeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ok || val != "hello" {
		t.Fatalf("got (%v, ok=%v), want (\"hello\", true)", val, ok)
	}
}

// TestConsumeVarsSurfacesRuntimeError verifies that an erroring configVars query is
// surfaced as an error rather than writing the error object into the config.
func TestConsumeVarsSurfacesRuntimeError(t *testing.T) {
	scope := vars.NewVariables(nil)
	scope.SetVar("n", "hello")

	var cfg struct {
		X int `yaml:"x"`
	}

	if err := scope.ConsumeVars(&cfg, map[string]string{"x": ".n + 1"}); err == nil {
		t.Fatal("expected an error for an erroring configVars query, got nil")
	}
}

// TestConsumeVarsValidQuery guards the happy path.
func TestConsumeVarsValidQuery(t *testing.T) {
	scope := vars.NewVariables(nil)
	scope.SetVar("num", 5)

	var cfg struct {
		X int `yaml:"x"`
	}

	if err := scope.ConsumeVars(&cfg, map[string]string{"x": ".num"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.X != 5 {
		t.Fatalf("cfg.X = %d, want 5", cfg.X)
	}
}
