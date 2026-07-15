package inboundauth

import "encoding/json"

// Sanitized is the client-safe view of a stored auth config: it reports what
// is configured without ever carrying secret or token material.
type Sanitized struct {
	Enabled   bool   `json:"enabled"`
	Scheme    Scheme `json:"scheme"`
	Preset    string `json:"preset,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	HasSecret bool   `json:"has_secret"`
}

// Sanitize parses a stored auth_config and strips credential material.
// Invalid configs sanitize to a disabled state rather than an error so a
// corrupt row can never leak its raw JSON through an API response.
func Sanitize(raw json.RawMessage) Sanitized {
	cfg, err := ParseSourceConfiguration(raw)
	if err != nil {
		return Sanitized{Scheme: SchemeNone}
	}
	return Sanitized{
		Enabled:   cfg.Scheme != SchemeNone,
		Scheme:    cfg.Scheme,
		Preset:    cfg.Preset,
		PublicKey: cfg.PublicKey,
		HasSecret: cfg.Secret != "" || cfg.Token != "",
	}
}
