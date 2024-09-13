package helper

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
