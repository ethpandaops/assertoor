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

	target := map[string]any{}

	err = json.Unmarshal(jsonData, &target)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal: %v\n%v", err, string(jsonData))
	}

	return target, nil
}
