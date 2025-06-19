package vars

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

type Variables struct {
	parentScope types.Variables
	varsMutex   sync.RWMutex
	varsMap     map[string]variableValue
	defaultsMap map[string]variableValue
	subScopes   map[string]types.Variables
}

type variableValue struct {
	isDefined bool
	value     interface{}
}

func newVariables(parentScope types.Variables) *Variables {
	return &Variables{
		parentScope: parentScope,
		varsMap:     map[string]variableValue{},
		defaultsMap: map[string]variableValue{},
		subScopes:   map[string]types.Variables{},
	}
}

func NewVariables(parentScope types.Variables) types.Variables {
	return newVariables(parentScope)
}

func (v *Variables) GetVar(name string) interface{} {
	v.varsMutex.RLock()
	subScope := v.subScopes[name]
	varValue := v.varsMap[name]
	defValue := v.defaultsMap[name]
	v.varsMutex.RUnlock()

	switch {
	case subScope != nil:
		return subScope.GetVarsMap(nil, false)
	case varValue.isDefined:
		return varValue.value
	case v.parentScope != nil:
		return v.parentScope.GetVar(name)
	case defValue.isDefined:
		return defValue.value
	default:
		return nil
	}
}

func (v *Variables) LookupVar(name string) (interface{}, bool) {
	v.varsMutex.RLock()
	subScope := v.subScopes[name]
	varValue := v.varsMap[name]
	defValue := v.defaultsMap[name]
	v.varsMutex.RUnlock()

	switch {
	case subScope != nil:
		return subScope.GetVarsMap(nil, false), true
	case varValue.isDefined:
		return varValue.value, true
	case v.parentScope != nil:
		return v.parentScope.LookupVar(name)
	case defValue.isDefined:
		return defValue.value, true
	default:
		return nil, false
	}
}

func (v *Variables) SetVar(name string, value interface{}) {
	v.varsMutex.Lock()
	v.varsMap[name] = variableValue{
		isDefined: true,
		value:     value,
	}
	v.varsMutex.Unlock()
}

func (v *Variables) SetDefaultVar(name string, value interface{}) {
	v.varsMutex.Lock()
	v.defaultsMap[name] = variableValue{
		isDefined: true,
		value:     value,
	}
	v.varsMutex.Unlock()
}

func (v *Variables) NewSubScope(name string) types.Variables {
	v.varsMutex.Lock()
	defer v.varsMutex.Unlock()

	subScope := newVariables(nil)
	v.subScopes[name] = subScope

	return subScope
}

func (v *Variables) GetSubScope(name string) types.Variables {
	var parentSubScope types.Variables

	if v.parentScope != nil {
		parentSubScope = v.parentScope.GetSubScope(name)
	}

	v.varsMutex.Lock()
	defer v.varsMutex.Unlock()

	subScope := v.subScopes[name]
	if subScope == nil {
		subScope = newVariables(parentSubScope)
		v.subScopes[name] = subScope
	}

	return subScope
}

func (v *Variables) SetSubScope(name string, subScope types.Variables) {
	v.varsMutex.Lock()
	defer v.varsMutex.Unlock()

	v.subScopes[name] = subScope
}

func (v *Variables) NewScope() types.Variables {
	return NewVariables(v)
}

func (v *Variables) ResolvePlaceholders(str string) string {
	r1 := regexp.MustCompile(`\${([^}]+)}`)
	str = r1.ReplaceAllStringFunc(str, func(m string) string {
		parts := r1.FindStringSubmatch(m)

		varValue, varFound := v.LookupVar(parts[1])
		if varFound {
			return fmt.Sprintf("%v", varValue)
		}

		return m
	})

	r2 := regexp.MustCompile(`\${{(.*?)}}`)
	str = r2.ReplaceAllStringFunc(str, func(m string) string {
		parts := r2.FindStringSubmatch(m)

		varValue, varFound, err := v.ResolveQuery(parts[1])
		if err != nil {
			return "?"
		}

		if varFound {
			return fmt.Sprintf("%v", varValue)
		}

		return "?"
	})

	return str
}

func (v *Variables) GetVarsMap(varsMap map[string]any, skipParent bool) map[string]any {
	if varsMap == nil {
		varsMap = map[string]any{}
	}

	v.varsMutex.RLock()
	for scopeName, subScope := range v.subScopes {
		_, exists := varsMap[scopeName]
		if exists {
			continue
		}

		varsMap[scopeName] = subScope.GetVarsMap(nil, skipParent)
	}

	for varName, varData := range v.varsMap {
		_, exists := varsMap[varName]
		if exists {
			continue
		}

		varsMap[varName] = varData.value
	}
	v.varsMutex.RUnlock()

	if v.parentScope != nil && !skipParent {
		varsMap = v.parentScope.GetVarsMap(varsMap, false)
	}

	v.varsMutex.RLock()
	for varName, varData := range v.defaultsMap {
		_, exists := varsMap[varName]
		if exists {
			continue
		}

		varsMap[varName] = varData.value
	}
	v.varsMutex.RUnlock()

	return varsMap
}

func (v *Variables) getGeneralizedVarsMap() (map[string]any, error) {
	varsMap := v.GetVarsMap(nil, false)

	// this is a bit hacky, but we're marshalling & unmarshalling varsMap here to generalize the types.
	// ie. []string should be a []interface{} of strings
	varsMapYaml, err := yaml.Marshal(&varsMap)
	if err != nil {
		return nil, fmt.Errorf("could not marshal scope variables: %v", err)
	}

	err = yaml.Unmarshal(varsMapYaml, varsMap)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal scope variables: %v", err)
	}

	return varsMap, nil
}

func (v *Variables) ResolveQuery(queryStr string) (val interface{}, ok bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	varsMap, err := v.getGeneralizedVarsMap()
	if err != nil {
		return nil, false, fmt.Errorf("could not get generalized variables: %v", err)
	}

	queryStr = fmt.Sprintf(".%v", queryStr)

	query, err := gojq.Parse(queryStr)
	if err != nil {
		return nil, false, fmt.Errorf("could not parse variable query '%v': %v", queryStr, err)
	}

	iter := query.RunWithContext(ctx, varsMap)

	val, ok = iter.Next()
	if !ok {
		// no query result, skip variable assignment
		return nil, false, nil
	}

	return val, true, nil
}

func (v *Variables) ConsumeVars(config interface{}, consumeMap map[string]string) error {
	applyMap := map[string]interface{}{}

	varsMap, err := v.getGeneralizedVarsMap()
	if err != nil {
		return fmt.Errorf("could not get generalized variables: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// now execute dynamic queries with gojq
	for cfgName, varQuery := range consumeMap {
		queryStr := fmt.Sprintf(".%v", varQuery)

		query, err2 := gojq.Parse(queryStr)
		if err2 != nil {
			return fmt.Errorf("could not parse variable query '%v': %v", queryStr, err2)
		}

		iter := query.RunWithContext(ctx, varsMap)

		val, ok := iter.Next()
		if !ok {
			// no query result, skip variable assignment
			continue
		}

		if v, ok := val.(float64); ok {
			val = NoScientificFloat64(v)
		}

		applyMap[cfgName] = val
	}

	// apply to config by generating a yaml, which is then parsed with the target config types.
	// that's a bit hacky again, but we don't have to care about types.
	applyYaml, err := yaml.Marshal(&applyMap)
	if err != nil {
		return fmt.Errorf("could not marshal dynamic config vars: %v", err)
	}

	err = yaml.Unmarshal(applyYaml, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal dynamic config vars: %v\n%v", err, string(applyYaml))
	}

	return nil
}

func (v *Variables) CopyVars(source types.Variables, copyMap map[string]string) error {
	for cfgName, varQuery := range copyMap {
		val, ok, err := source.ResolveQuery(varQuery)
		if err != nil {
			return err
		}

		if !ok {
			continue
		}

		v.SetVar(cfgName, val)
	}

	return nil
}
