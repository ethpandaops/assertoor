package vars

import (
	"sync"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
)

type Variables struct {
	parentScope types.Variables
	varsMutex   sync.RWMutex
	varsMap     map[string]variableValue
}

type variableValue struct {
	isDefined bool
	value     interface{}
}

func NewVariables(parentScope types.Variables) types.Variables {
	return &Variables{
		parentScope: parentScope,
		varsMap:     map[string]variableValue{},
	}
}

func (v *Variables) GetVar(name string) interface{} {
	v.varsMutex.RLock()
	varValue := v.varsMap[name]
	v.varsMutex.RUnlock()

	if varValue.isDefined {
		return varValue.value
	} else if v.parentScope != nil {
		return v.parentScope.GetVar(name)
	}

	return nil
}

func (v *Variables) SetVar(name string, value interface{}) {
	v.varsMutex.Lock()
	v.varsMap[name] = variableValue{
		isDefined: true,
		value:     value,
	}
	v.varsMutex.Unlock()
}
