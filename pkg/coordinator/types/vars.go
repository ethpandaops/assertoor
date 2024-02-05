package types

type Variables interface {
	GetVar(name string) interface{}
	LookupVar(name string) (interface{}, bool)
	ResolveQuery(query string) (interface{}, bool, error)
	SetVar(name string, value interface{})
	NewScope() Variables
	GetVarsMap() map[string]any
	ResolvePlaceholders(str string) string
	ConsumeVars(config interface{}, consumeMap map[string]string) error
	CopyVars(source Variables, copyMap map[string]string)
}
