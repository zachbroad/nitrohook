package inboundauth

import (
	"net/http"
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

// Temporary stubs for schemes implemented in subsequent tasks
func verifyHMAC(Config, http.Header, []byte, time.Time) error { return ErrUnsupportedScheme }
func verifyToken(Config, http.Header) error                   { return ErrUnsupportedScheme }
func verifyBearer(Config, http.Header) error                  { return ErrUnsupportedScheme }
func verifyEd25519(Config, http.Header, []byte, time.Time) error { return ErrUnsupportedScheme }
