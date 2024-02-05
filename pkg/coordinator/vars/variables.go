package vars

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/itchyny/gojq"
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

func (v *Variables) GetVarsMap() map[string]any {
	var varsMap map[string]any

	if v.parentScope != nil {
		varsMap = v.parentScope.GetVarsMap()
	} else {
		varsMap = map[string]any{}
	}

	for varName, varData := range v.varsMap {
		varsMap[varName] = varData.value
	}

	return varsMap
}

//nolint:gocritic // ignore
func (v *Variables) ResolveQuery(queryStr string) (interface{}, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	varsMap := v.GetVarsMap()
	queryStr = fmt.Sprintf(".%v", queryStr)

	query, err := gojq.Parse(queryStr)
	if err != nil {
		return nil, false, fmt.Errorf("could not parse variable query '%v': %v", queryStr, err)
	}

	iter := query.RunWithContext(ctx, varsMap)

	val, ok := iter.Next()
	if !ok {
		// no query result, skip variable assignment
		return nil, false, nil
	}

	return val, true, nil
}

func (v *Variables) ConsumeVars(config interface{}, consumeMap map[string]string) error {
	applyMap := map[string]interface{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	varsMap := v.GetVarsMap()

	for cfgName, varQuery := range consumeMap {
		queryStr := fmt.Sprintf(".%v", varQuery)

		query, err := gojq.Parse(queryStr)
		if err != nil {
			return fmt.Errorf("could not parse variable query '%v': %v", queryStr, err)
		}

		iter := query.RunWithContext(ctx, varsMap)

		val, ok := iter.Next()
		if !ok {
			// no query result, skip variable assignment
			continue
		}

		applyMap[cfgName] = val
	}

	// apply to config by generating a yaml, which is then parsed with the target config types.
	// that's a bit hacky, but we don't have to care about types.
	applyYaml, err := yaml.Marshal(&applyMap)
	if err != nil {
		return fmt.Errorf("could not marshal dynamic config vars")
	}

	err = yaml.Unmarshal(applyYaml, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal dynamic config vars")
	}

	return nil
}

func (v *Variables) CopyVars(source types.Variables, copyMap map[string]string) {
	for targetName, varName := range copyMap {
		varValue, varFound := source.LookupVar(varName)
		if !varFound {
			continue
		}

		v.SetVar(targetName, varValue)
	}
}
