package inboundauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"net/http"
	"strings"
	"time"
)

// Verify returns nil if the request is authentic under cfg, or a sentinel error.
func Verify(cfg Config, header http.Header, rawBody []byte, now time.Time) error {
	switch cfg.Scheme {
	case SchemeNone:
		return nil
	case SchemeHMAC:
		return verifyHMAC(cfg, header, rawBody, now)
	case SchemeToken:
		return verifyToken(cfg, header)
	case SchemeBearer:
		return verifyBearer(cfg, header)
	case SchemeEd25519:
		return verifyEd25519(cfg, header, rawBody, now)
	default:
		return ErrUnsupportedScheme
	}
}

func verifyHMAC(cfg Config, header http.Header, rawBody []byte, now time.Time) error {
	sig, err := extractSignature(cfg, header)
	if err != nil {
		return err
	}
	signed, err := buildSignedPayload(cfg, header, rawBody, now)
	if err != nil {
		return err
	}
	expected := computeHMAC(cfg, signed)
	got, err := decodeSig(cfg, sig)
	if err != nil {
		return ErrBadSignature
	}
	if !hmac.Equal(expected, got) {
		return ErrBadSignature
	}
	return nil
}

func hasher(algo string) func() hash.Hash {
	if algo == "sha1" {
		return sha1.New
	}
	return sha256.New
}

// computeHMAC returns the raw (un-encoded) HMAC of signed.
func computeHMAC(cfg Config, signed []byte) []byte {
	mac := hmac.New(hasher(cfg.Algo), []byte(cfg.Secret))
	mac.Write(signed)
	return mac.Sum(nil)
}

// decodeSig decodes a received signature string into raw bytes per cfg.Encoding.
func decodeSig(cfg Config, sig string) ([]byte, error) {
	if cfg.Encoding == "base64" {
		return base64.StdEncoding.DecodeString(sig)
	}
	return hex.DecodeString(sig)
}

// extractSignature returns the raw signature value using the configured parser.
// The "plain" parser handles the raw-body family (GitHub, Forgejo, Shopify).
// Other parsers are added in Task 4.
func extractSignature(cfg Config, header http.Header) (string, error) {
	switch cfg.SigParser {
	case "plain", "":
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return "", ErrMissingSignature
		}
		return strings.TrimPrefix(raw, cfg.SigPrefix), nil
	default:
		return extractSignatureAdvanced(cfg, header)
	}
}

// buildSignedPayload constructs the byte string that gets HMAC'd.
// "raw_body" is the whole body. Other templates are added in Task 4.
func buildSignedPayload(cfg Config, header http.Header, rawBody []byte, now time.Time) ([]byte, error) {
	switch cfg.Template {
	case "raw_body", "":
		return rawBody, nil
	default:
		return buildSignedPayloadAdvanced(cfg, header, rawBody, now)
	}
}

// extractSignatureAdvanced handles non-"plain" signature parsers (kv-comma,
// space-list, slack). Implemented in Task 4.
func extractSignatureAdvanced(Config, http.Header) (string, error) {
	return "", ErrUnsupportedScheme
}

// buildSignedPayloadAdvanced handles non-"raw_body" signing templates
// (ts.body, id.ts.body, slack_v0). Implemented in Task 4.
func buildSignedPayloadAdvanced(Config, http.Header, []byte, time.Time) ([]byte, error) {
	return nil, ErrUnsupportedScheme
}

// Temporary stubs for schemes implemented in subsequent tasks
func verifyToken(Config, http.Header) error                      { return ErrUnsupportedScheme }
func verifyBearer(Config, http.Header) error                     { return ErrUnsupportedScheme }
func verifyEd25519(Config, http.Header, []byte, time.Time) error { return ErrUnsupportedScheme }
