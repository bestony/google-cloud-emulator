package dns

import "net/http"

// Register mounts the Cloud DNS REST handler on mux under /dns/v1/.
func Register(mux *http.ServeMux, state *State) {
	mux.Handle("/dns/v1/", NewHandler(state))
}
