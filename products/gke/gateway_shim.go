package gke

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

var (
	clustersCollectionRe = regexp.MustCompile(`^/v1/projects/[^/]+/(?:zones|locations)/[^/]+/clusters$`)
	clusterResourceRe    = regexp.MustCompile(`^/v1/projects/[^/]+/(?:zones|locations)/[^/]+/clusters/[^/]+$`)
)

// WrapGateway adapts GKE's `body: "*"` transcoding to real GKE REST
// semantics. Google's frontend maps the request body to the method
// signature's body field (cluster on create, update on update), while
// grpc-gateway decodes a "*" body into the whole request message. Bare
// resource bodies (what gcloud and the Discovery-based clients send) are
// rewritten into the wrapped form before the gateway sees them; bodies
// that already carry the wrapper key pass through untouched.
func WrapGateway(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var key string
		switch {
		case r.Method == http.MethodPost && clustersCollectionRe.MatchString(r.URL.Path):
			key = "cluster"
		case r.Method == http.MethodPut && clusterResourceRe.MatchString(r.URL.Path):
			key = "update"
		}
		if key == "" || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil || len(body) == 0 {
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
			return
		}
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(body, &probe); err != nil {
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := probe[key]; ok {
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
			return
		}
		wrapped, err := json.Marshal(map[string]json.RawMessage{key: body})
		if err != nil {
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(wrapped))
		r.ContentLength = int64(len(wrapped))
		r.Header.Set("Content-Length", strconv.Itoa(len(wrapped)))
		next.ServeHTTP(w, r)
	})
}
