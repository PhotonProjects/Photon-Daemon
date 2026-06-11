package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"https://github.com/PhotonProjects/Photon-Panel"
	"github.com/Maj-Studios/Photon-Daemon/internal/remote"
	"github.com/Maj-Studios/Photon-Daemon/internal/server"
)

// ServerStore gère le stockage des serveurs.
type ServerStore interface {
	Get(uuid string) *server.Server
	List() []*server.Server
	Add(s *server.Server)
	Remove(uuid string)
}

// Router configure les routes HTTP de l'API.
type Router struct {
	router      *mux.Router
	store       ServerStore
	remote      *remote.Client
}

// NewRouter crée un nouveau routeur API.
func NewRouter(store ServerStore, remoteClient *remote.Client) *Router {
	r := &Router{
		router: mux.NewRouter(),
		store:  store,
		remote: remoteClient,
	}
	r.registerRoutes()
	return r
}

func (r *Router) registerRoutes() {
	api := r.router.PathPrefix("/api").Subrouter()
	api.Use(authMiddleware)

	// Servers
	api.HandleFunc("/servers", r.handleListServers).Methods("GET")
	api.HandleFunc("/servers", r.handleCreateServer).Methods("POST")
	api.HandleFunc("/servers/{uuid}", r.handleGetServer).Methods("GET")
	api.HandleFunc("/servers/{uuid}", r.handleDeleteServer).Methods("DELETE")
	api.HandleFunc("/servers/{uuid}/power", r.handlePowerAction).Methods("POST")
	api.HandleFunc("/servers/{uuid}/install", r.handleInstall).Methods("POST")
	api.HandleFunc("/servers/{uuid}/sync", r.handleSync).Methods("POST")
}

// ServeHTTP implémente http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.router.ServeHTTP(w, req)
}

// ListenAndServe démarre le serveur HTTP.
func ListenAndServe(ctx context.Context, store ServerStore, remoteClient *remote.Client) error {
	cfg := config.Get()
	router := NewRouter(store, remoteClient)

	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}
