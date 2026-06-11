package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Maj-Studios/Photon-Daemon/internal/remote"
	"github.com/Maj-Studios/Photon-Daemon/internal/server"
)

type handler struct {
	store  ServerStore
	remote *remote.Client
}

// ----- Servers -----

func (r *Router) handleListServers(w http.ResponseWriter, req *http.Request) {
	servers := r.store.List()
	writeJSON(w, http.StatusOK, servers)
}

func (r *Router) handleCreateServer(w http.ResponseWriter, req *http.Request) {
	var cfg server.ServerConfig
	if err := json.NewDecoder(req.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	uuid := req.URL.Query().Get("uuid")
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "uuid is required")
		return
	}

	s, err := server.New(uuid, cfg, r.remote)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	r.store.Add(s)
	writeJSON(w, http.StatusCreated, s)
}

func (r *Router) handleGetServer(w http.ResponseWriter, req *http.Request) {
	uuid := mux.Vars(req)["uuid"]
	s := r.store.Get(uuid)
	if s == nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (r *Router) handleDeleteServer(w http.ResponseWriter, req *http.Request) {
	uuid := mux.Vars(req)["uuid"]
	s := r.store.Get(uuid)
	if s == nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	s.Cancel()
	r.store.Remove(uuid)
	writeJSON(w, http.StatusNoContent, nil)
}

// ----- Power -----

func (r *Router) handlePowerAction(w http.ResponseWriter, req *http.Request) {
	uuid := mux.Vars(req)["uuid"]
	s := r.store.Get(uuid)
	if s == nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.HandlePower(server.PowerAction(body.Action)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ----- Install -----

func (r *Router) handleInstall(w http.ResponseWriter, req *http.Request) {
	uuid := mux.Vars(req)["uuid"]
	s := r.store.Get(uuid)
	if s == nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	go func() {
		if err := s.Install(); err != nil {
			fmt.Printf("server %s install failed: %v\n", uuid, err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "installing"})
}

// ----- Sync -----

func (r *Router) handleSync(w http.ResponseWriter, req *http.Request) {
	uuid := mux.Vars(req)["uuid"]
	s := r.store.Get(uuid)
	if s == nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ----- Utilitaires -----

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
