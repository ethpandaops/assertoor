package vars

import (
	"encoding/json"
	"fmt"
)

func GeneralizeData(source interface{}) (interface{}, error) {
	jsonData, err := json.Marshal(&source)
	if err != nil {
		return nil, fmt.Errorf("could not marshal: %v", err)
	}

	var target interface{}

	err = json.Unmarshal(jsonData, &target)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal: %v\n%v", err, string(jsonData))
	}

	return target, nil
}

type NoScientificFloat64 float64

func (n NoScientificFloat64) MarshalJSON() ([]byte, error) {
	return []byte(n.getTrimmedFloat()), nil
}

func (n NoScientificFloat64) MarshalYAML() (interface{}, error) {
	return n.getTrimmedFloat(), nil
}

func (n NoScientificFloat64) getTrimmedFloat() string {
	str := fmt.Sprintf("%f", n)

	// Trim trailing zeros
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] == '0' {
			str = str[:i]
		} else {
			break
		}
	}

	// Trim trailing dot
	if str[len(str)-1] == '.' {
		str = str[:len(str)-1]
	}

	return str
}
