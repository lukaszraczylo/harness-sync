package render

import "github.com/pelletier/go-toml/v2"

func TOML(v any) ([]byte, error) {
	return toml.Marshal(v)
}
