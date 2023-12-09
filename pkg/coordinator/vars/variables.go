package vars

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"gopkg.in/yaml.v2"
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

func (v *Variables) LookupVar(name string) (interface{}, bool) {
	v.varsMutex.RLock()
	varValue := v.varsMap[name]
	v.varsMutex.RUnlock()

	if varValue.isDefined {
		return varValue.value, true
	} else if v.parentScope != nil {
		return v.parentScope.LookupVar(name)
	}

	return nil, false
}

func (v *Variables) SetVar(name string, value interface{}) {
	v.varsMutex.Lock()
	v.varsMap[name] = variableValue{
		isDefined: true,
		value:     value,
	}
	v.varsMutex.Unlock()
}

func (v *Variables) NewScope() types.Variables {
	return NewVariables(v)
}

func (v *Variables) ResolvePlaceholders(str string) string {
	r := regexp.MustCompile(`\${([^}]+)}`)

	return r.ReplaceAllStringFunc(str, func(m string) string {
		parts := r.FindStringSubmatch(m)

		varValue, varFound := v.LookupVar(parts[1])
		if varFound {
			return fmt.Sprintf("%v", varValue)
		}
		return m
	})
}

func (v *Variables) ConsumeVars(config interface{}, consumeMap map[string]string) error {
	applyMap := map[string]interface{}{}

	for cfgName, varName := range consumeMap {
		varValue, varFound := v.LookupVar(varName)
		if !varFound {
			continue
		}

		applyMap[cfgName] = varValue
	}

	// apply to confiy by generating a yaml, which is then parsed with the config types
	// dirty hack, but we don't have to care about types this way
	applyYaml, err := yaml.Marshal(&applyMap)
	if err != nil {
		return fmt.Errorf("could not marshal dynamic config vars")
	}

	fmt.Printf("merge config: %v", string(applyYaml))

	err = yaml.Unmarshal(applyYaml, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal dynamic config vars")
	}

	return nil
}
