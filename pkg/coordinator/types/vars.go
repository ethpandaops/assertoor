package types

type Variables interface {
	GetVar(name string) interface{}
	LookupVar(name string) (interface{}, bool)
	SetVar(name string, value interface{})
	NewScope() Variables
	ResolvePlaceholders(str string) string
	ConsumeVars(config interface{}, consumeMap map[string]string) error
}
