package helper

import "gopkg.in/yaml.v3"

type IRawMessage interface {
	UnmarshalYAML(func(interface{}) error) error
	Unmarshal(interface{}) error
}

type RawMessage struct {
	unmarshal func(interface{}) error
}

// NewRawMessage builds a RawMessage from any Go value by routing through
// YAML. Useful for programmatically constructing TaskOptions.Config from
// a typed config struct.
func NewRawMessage(config any) *RawMessage {
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return nil
	}

	raw := RawMessage{}
	if err := yaml.Unmarshal(configYaml, &raw); err != nil {
		return nil
	}

	return &raw
}

func (r *RawMessage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	r.unmarshal = unmarshal
	return nil
}

func (r *RawMessage) MarshalYAML() (interface{}, error) {
	data := map[string]interface{}{}
	if err := r.unmarshal(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func (r *RawMessage) Unmarshal(v interface{}) error {
	return r.unmarshal(v)
}

type RawMessageMasked struct {
	unmarshal func(interface{}) error
}

func (r *RawMessageMasked) UnmarshalYAML(unmarshal func(interface{}) error) error {
	r.unmarshal = unmarshal
	return nil
}

func (r *RawMessageMasked) Unmarshal(v interface{}) error {
	return r.unmarshal(v)
}
