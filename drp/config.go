package drp

import "encoding/json"

type Config struct {
	Address  string          `json:"address,omitempty"`
	Path     string          `json:"path,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}
