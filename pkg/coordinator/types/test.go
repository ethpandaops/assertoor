package types

import "context"

type Test interface {
	Validate() error
	Run(ctx context.Context) error
	Name() string
	Percent() float64
}
