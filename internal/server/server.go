package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"stackpilot/internal/dockpilot"
	"stackpilot/internal/stack"
)

type deployRequest struct {
	YAML string `json:"yaml"`
}

type removeRequest struct {
	YAML    string `json:"yaml"`
	Volumes bool   `json:"volumes"`
}

type serviceResult struct {
	Name      string `json:"name"`
	Container string `json:"container"`
	State     string `json:"state"`
	Ports     string `json:"ports"`
	Running   bool   `json:"running"`
}

type deployResponse struct {
	Stack    string          `json:"stack"`
	Services []serviceResult `json:"services"`
}

type errResponse struct {
	Error string `json:"error"`
}

// Run starts the HTTP server on addr, forwarding stack operations to dockpilotURL.
func Run(addr, dockpilotURL string) error {
	client := dockpilot.New(dockpilotURL)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /v1/stacks/deploy", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "reading body: "+err.Error())
			return
		}
		var req deployRequest
		if err := json.Unmarshal(body, &req); err != nil || req.YAML == "" {
			writeError(w, http.StatusBadRequest, `body must be JSON with a "yaml" field`)
			return
		}
		s, err := stack.ParseBytes([]byte(req.YAML), "<api>")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := stack.Validate(s); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		if err := stack.Deploy(ctx, client, s); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		statuses, _ := stack.Status(ctx, client, s)
		resp := deployResponse{Stack: s.Name}
		for _, st := range statuses {
			resp.Services = append(resp.Services, serviceResult{
				Name:      st.Name,
				Container: st.Container,
				State:     st.State,
				Ports:     st.Ports,
				Running:   st.Running,
			})
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/stacks/remove", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "reading body: "+err.Error())
			return
		}
		var req removeRequest
		if err := json.Unmarshal(body, &req); err != nil || req.YAML == "" {
			writeError(w, http.StatusBadRequest, `body must be JSON with a "yaml" field`)
			return
		}
		s, err := stack.ParseBytes([]byte(req.YAML), "<api>")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()
		if err := stack.Remove(ctx, client, s, req.Volumes); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"stack": s.Name, "status": "removed"})
	})

	log.Printf("stackpilot API listening on http://%s  →  dockpilot at %s", addr, dockpilotURL)
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	log.Printf("error %d: %s", status, msg)
	writeJSON(w, status, errResponse{Error: msg})
}
