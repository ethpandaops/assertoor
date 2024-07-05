package vars

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
)

type ScopeFilter struct {
	vars types.Variables
}

func NewScopeFilter(vars types.Variables) types.Variables {
	return &ScopeFilter{
		vars: vars,
	}
}

func (v *ScopeFilter) GetVar(name string) interface{} {
	return v.vars.GetVar(name)
}

func (v *ScopeFilter) LookupVar(name string) (interface{}, bool) {
	return v.vars.LookupVar(name)
}

func (v *ScopeFilter) SetVar(name string, value interface{}) {
	v.vars.SetVar(name, value)
}

func (v *ScopeFilter) NewSubScope(name string) types.Variables {
	return v.vars.NewSubScope(name)
}

func (v *ScopeFilter) GetSubScope(name string) types.Variables {
	return v.vars.GetSubScope(name)
}

func (v *ScopeFilter) SetSubScope(name string, subScope types.Variables) {
	v.vars.SetSubScope(name, subScope)
}

func (v *ScopeFilter) NewScope() types.Variables {
	return v.vars.NewScope()
}

func (v *ScopeFilter) ResolvePlaceholders(str string) string {
	return v.vars.ResolvePlaceholders(str)
}

func (v *ScopeFilter) GetVarsMap(varsMap map[string]any, _ bool) map[string]any {
	return v.vars.GetVarsMap(varsMap, true)
}

func (v *ScopeFilter) ResolveQuery(queryStr string) (value interface{}, found bool, err error) {
	return v.vars.ResolveQuery(queryStr)
}

func (v *ScopeFilter) ConsumeVars(config interface{}, consumeMap map[string]string) error {
	return v.vars.ConsumeVars(config, consumeMap)
}

func (v *ScopeFilter) CopyVars(source types.Variables, copyMap map[string]string) error {
	return v.vars.CopyVars(source, copyMap)
}
