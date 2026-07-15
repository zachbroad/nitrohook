// Package inboundauth verifies the authenticity of incoming webhook requests
// against per-source configuration covering the major provider signing schemes.
package inboundauth

import (
	"encoding/json"
	"errors"
)

type Scheme string

const (
	SchemeNone    Scheme = "none"
	SchemeHMAC    Scheme = "hmac"
	SchemeBearer  Scheme = "bearer"
	SchemeToken   Scheme = "token"
	SchemeEd25519 Scheme = "ed25519"
)

// SourceConfiguration is the decoded per-source auth configuration (from sources.auth_config).
type SourceConfiguration struct {
	Scheme Scheme `json:"scheme"`
	Preset string `json:"preset,omitempty"`

	// HMAC axes.
	Algo      string `json:"algo,omitempty"`       // "sha256" | "sha1"
	Encoding  string `json:"encoding,omitempty"`   // "hex" | "base64"
	SigHeader string `json:"sig_header,omitempty"` // header carrying the signature
	SigParser string `json:"sig_parser,omitempty"` // "plain" | "kv-comma" | "space-list" | "slack"
	SigPrefix string `json:"sig_prefix,omitempty"` // stripped by the "plain" parser (e.g. "sha256=")
	Template  string `json:"template,omitempty"`   // "raw_body" | "ts.body" | "id.ts.body" | "slack_v0"

	// Timestamp / replay (used by ts-bound templates).
	TSHeader     string `json:"ts_header,omitempty"`
	IDHeader     string `json:"id_header,omitempty"`
	TSToleranceS int    `json:"ts_tolerance_secs,omitempty"`

	// Secrets (plaintext at rest).
	Secret    string `json:"secret,omitempty"`       // HMAC shared secret
	Token     string `json:"token,omitempty"`        // bearer / plain-token expected value
	TokenHdr  string `json:"token_header,omitempty"` // header for "token" scheme (e.g. X-Gitlab-Token)
	PublicKey string `json:"public_key,omitempty"`   // ed25519 public key, hex-encoded
}

var (
	ErrMissingSignature        = errors.New("missing signature header")
	ErrBadSignature            = errors.New("signature mismatch")
	ErrMissingTimestamp        = errors.New("missing timestamp header")
	ErrTimestampOutOfTolerance = errors.New("timestamp outside tolerance window")
	ErrUnsupportedScheme       = errors.New("unsupported auth scheme")
	ErrMissingSecret           = errors.New("scheme configured without a secret/token")
)

// ParseSourceConfiguration decodes sources.auth_config. Nil/empty means no authentication.
func ParseSourceConfiguration(raw json.RawMessage) (SourceConfiguration, error) {
	if len(raw) == 0 {
		return SourceConfiguration{Scheme: SchemeNone}, nil
	}
	var cfg SourceConfiguration
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return SourceConfiguration{}, err
	}
	if cfg.Scheme == "" {
		cfg.Scheme = SchemeNone
	}
	return cfg, nil
}

// FailureReason maps a verification error to a short, low-cardinality metric label.
func FailureReason(err error) string {
	switch {
	case errors.Is(err, ErrMissingSignature):
		return "missing_signature"
	case errors.Is(err, ErrBadSignature):
		return "bad_signature"
	case errors.Is(err, ErrMissingTimestamp):
		return "missing_timestamp"
	case errors.Is(err, ErrTimestampOutOfTolerance):
		return "timestamp"
	case errors.Is(err, ErrUnsupportedScheme):
		return "unsupported"
	case errors.Is(err, ErrMissingSecret):
		return "missing_secret"
	default:
		return "error"
	}
}
