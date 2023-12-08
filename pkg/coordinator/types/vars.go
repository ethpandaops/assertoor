package types

type Variables interface {
	GetVar(name string) interface{}
	SetVar(name string, value interface{})
}
