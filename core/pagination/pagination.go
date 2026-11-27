// Package pagination implements the page-token convention shared by Google
// list APIs: an opaque token that encodes the offset into the result set.
package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

// Encode returns the page token for the given offset.
func Encode(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// Decode parses a page token. An empty token yields offset 0.
func Decode(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid page token")
	}
	offset, err := strconv.Atoi(string(raw))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid page token")
	}
	return offset, nil
}

// PageToken returns the next-page token for offset+pageSize when more results
// remain, or "" when the page is the last one.
func PageToken(offset, pageSize, remaining int) string {
	if offset+pageSize < remaining {
		return Encode(offset + pageSize)
	}
	return ""
}

// EffectivePageSize applies the API default when pageSize is unset.
func EffectivePageSize(pageSize int32, def int) int {
	if pageSize <= 0 {
		return def
	}
	return int(pageSize)
}
