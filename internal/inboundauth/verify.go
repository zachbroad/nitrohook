package inboundauth

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"net/http"
	"strconv"
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
	sigs, err := extractSignatures(cfg, header)
	if err != nil {
		return err
	}
	signed, err := buildSignedPayload(cfg, header, rawBody, now)
	if err != nil {
		return err
	}
	expected := computeHMAC(cfg, signed)
	for _, sig := range sigs {
		got, err := decodeSig(cfg, sig)
		if err != nil {
			continue
		}
		if hmac.Equal(expected, got) {
			return nil
		}
	}
	return ErrBadSignature
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

// extractSignatures returns the candidate raw signature values using the
// configured parser. The "plain" parser handles the raw-body family (GitHub,
// Forgejo, Shopify). "kv-comma" and "space-list" parsers may yield multiple
// candidates (e.g. rotated secrets); each is checked in verifyHMAC.
func extractSignatures(cfg Config, header http.Header) ([]string, error) {
	switch cfg.SigParser {
	case "plain", "":
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		return []string{strings.TrimPrefix(raw, cfg.SigPrefix)}, nil
	case "slack":
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		return []string{strings.TrimPrefix(raw, "v0=")}, nil
	case "kv-comma": // Stripe: "t=...,v1=...,v0=..."
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		var out []string
		for _, part := range strings.Split(raw, ",") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) == 2 && kv[0] == "v1" {
				out = append(out, kv[1])
			}
		}
		if len(out) == 0 {
			return nil, ErrMissingSignature
		}
		return out, nil
	case "space-list": // Svix: "v1,<b64> v1,<b64>"
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		var out []string
		for _, tok := range strings.Fields(raw) {
			if i := strings.IndexByte(tok, ','); i >= 0 {
				out = append(out, tok[i+1:])
			}
		}
		if len(out) == 0 {
			return nil, ErrMissingSignature
		}
		return out, nil
	default:
		return nil, ErrUnsupportedScheme
	}
}

// buildSignedPayload constructs the byte string that gets HMAC'd.
// "raw_body" is the whole body. The timestamp-bound templates (ts.body,
// id.ts.body, slack_v0) enforce a tolerance window against now.
func buildSignedPayload(cfg Config, header http.Header, rawBody []byte, now time.Time) ([]byte, error) {
	switch cfg.Template {
	case "raw_body", "":
		return rawBody, nil
	case "ts.body", "id.ts.body", "slack_v0":
		ts, err := extractTimestamp(cfg, header)
		if err != nil {
			return nil, err
		}
		if err := checkTolerance(ts, cfg.TSToleranceS, now); err != nil {
			return nil, err
		}
		switch cfg.Template {
		case "ts.body":
			return []byte(ts + "." + string(rawBody)), nil
		case "slack_v0":
			return []byte("v0:" + ts + ":" + string(rawBody)), nil
		case "id.ts.body":
			id := header.Get(cfg.IDHeader)
			return []byte(id + "." + ts + "." + string(rawBody)), nil
		}
	}
	return nil, ErrUnsupportedScheme
}

// extractTimestamp returns the timestamp string for a timestamp-bound
// template. Most schemes (Slack, Svix) carry it in a dedicated header
// (cfg.TSHeader). Stripe's kv-comma scheme has no separate timestamp
// header — it embeds "t=<ts>" alongside the signature in cfg.SigHeader —
// so that's used as a fallback when TSHeader isn't configured.
func extractTimestamp(cfg Config, header http.Header) (string, error) {
	if cfg.TSHeader != "" {
		ts := header.Get(cfg.TSHeader)
		if ts == "" {
			return "", ErrMissingTimestamp
		}
		return ts, nil
	}
	if cfg.SigParser == "kv-comma" {
		raw := header.Get(cfg.SigHeader)
		for _, part := range strings.Split(raw, ",") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) == 2 && kv[0] == "t" {
				return kv[1], nil
			}
		}
	}
	return "", ErrMissingTimestamp
}

// checkTolerance rejects timestamps outside the configured window. A
// non-positive tolerance disables the check.
func checkTolerance(tsStr string, toleranceS int, now time.Time) error {
	if toleranceS <= 0 {
		return nil
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return ErrMissingTimestamp
	}
	diff := now.Unix() - ts
	if diff < 0 {
		diff = -diff
	}
	if diff > int64(toleranceS) {
		return ErrTimestampOutOfTolerance
	}
	return nil
}

func verifyToken(cfg Config, header http.Header) error {
	got := header.Get(cfg.TokenHdr)
	if got == "" {
		return ErrMissingSignature
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(cfg.Token)) != 1 {
		return ErrBadSignature
	}
	return nil
}

func verifyBearer(cfg Config, header http.Header) error {
	raw := header.Get("Authorization")
	if raw == "" {
		return ErrMissingSignature
	}
	got := strings.TrimPrefix(raw, "Bearer ")
	if subtle.ConstantTimeCompare([]byte(got), []byte(cfg.Token)) != 1 {
		return ErrBadSignature
	}
	return nil
}

func verifyEd25519(cfg Config, header http.Header, rawBody []byte, now time.Time) error {
	sigHex := header.Get(cfg.SigHeader)
	if sigHex == "" {
		return ErrMissingSignature
	}
	ts := header.Get(cfg.TSHeader)
	if ts == "" {
		return ErrMissingTimestamp
	}
	if cfg.TSToleranceS > 0 {
		if err := checkTolerance(ts, cfg.TSToleranceS, now); err != nil {
			return err
		}
	}
	pub, err := hex.DecodeString(cfg.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return ErrBadSignature
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return ErrBadSignature
	}
	msg := append([]byte(ts), rawBody...)
	if !ed25519.Verify(ed25519.PublicKey(pub), msg, sig) {
		return ErrBadSignature
	}
	return nil
}
