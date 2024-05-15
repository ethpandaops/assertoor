package helper

import (
	"fmt"
	"math/big"
)

type BigInt struct {
	Value big.Int
}

func (b BigInt) MarshalJSON() ([]byte, error) {
	return []byte(b.Value.String()), nil
}

func (b *BigInt) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}
	var z big.Int
	_, ok := z.SetString(string(p), 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", p)
	}
	b.Value = z
	return nil
}

func (b *BigInt) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string

	err := unmarshal(&value)
	if err != nil {
		return err
	}

	if value == "null" {
		return nil
	}
	var z big.Int
	_, ok := z.SetString(value, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", &b.Value)
	}
	b.Value = z
	return nil
}

func (b *BigInt) MarshalYAML() (interface{}, error) {
	return b.Value.String(), nil
}
