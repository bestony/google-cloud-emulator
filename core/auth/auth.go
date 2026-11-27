// Package auth extracts a caller identity from incoming requests. The emulator
// accepts any bearer token (or none); the identity is only used where a product
// chooses to record or authorize it.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// DefaultEmail is the identity used when no token is present or the token
// carries no recognizable email claim.
const DefaultEmail = "dev@localhost"

// Principal identifies the caller.
type Principal struct {
	Email string
}

// FromContext reads the Authorization metadata of a gRPC/gateway request.
func FromContext(ctx context.Context) Principal {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{Email: DefaultEmail}
	}
	for _, v := range md.Get("authorization") {
		if p, ok := principalFromBearer(v); ok {
			return p
		}
	}
	for _, v := range md.Get("x-goog-authenticated-user-email") {
		return Principal{Email: v}
	}
	return Principal{Email: DefaultEmail}
}

// FromRequest reads the Authorization header of a plain REST request.
func FromRequest(r *http.Request) Principal {
	if p, ok := principalFromBearer(r.Header.Get("Authorization")); ok {
		return p
	}
	return Principal{Email: DefaultEmail}
}

func principalFromBearer(header string) (Principal, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return Principal{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return Principal{Email: DefaultEmail}, true
	}
	// Best effort: read the email claim of a JWT without verifying it.
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		if payload, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims struct {
				Email string `json:"email"`
				Sub   string `json:"sub"`
			}
			if json.Unmarshal(payload, &claims) == nil {
				if claims.Email != "" {
					return Principal{Email: claims.Email}, true
				}
				if claims.Sub != "" {
					return Principal{Email: claims.Sub}, true
				}
			}
		}
	}
	// Opaque token (e.g. "ya29...." or anything else): accept it, no identity.
	return Principal{Email: DefaultEmail}, true
}
