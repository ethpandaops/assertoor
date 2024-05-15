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
