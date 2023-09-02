package config

type Application struct {
	// name is used as key to identify the directives
	Name string `json:"name"`

	// directives
	Directives string `json:"directives"`
}
