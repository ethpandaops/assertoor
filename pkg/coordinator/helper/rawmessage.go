package helper

type IRawMessage interface {
	UnmarshalYAML(func(interface{}) error) error
	Unmarshal(interface{}) error
}

type RawMessage struct {
	unmarshal func(interface{}) error
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
