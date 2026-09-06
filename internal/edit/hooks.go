package edit

// Test hooks — production defaults call the real implementations.
var (
	SetJSONHook   = setJSONImpl
	SetJSONCHook  = setJSONCImpl
	SetYAMLHook   = setYAMLImpl
	SetTOMLHook   = setTOMLImpl
	SetDotEnvHook = setDotEnvImpl
)
