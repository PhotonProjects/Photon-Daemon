package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Maj-Studios/Photon-Daemon/api"
	"https://github.com/PhotonProjects/Photon-Panel"
	photonDocker "github.com/Maj-Studios/Photon-Daemon/internal/docker"
	"github.com/Maj-Studios/Photon-Daemon/internal/remote"
)

func main() {
	configPath := flag.String("config", "/etc/photon/config.yml", "path to config file")
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialiser le client Docker
	log.Println("connecting to Docker daemon...")
	if _, err := photonDocker.NewClient(); err != nil {
		log.Fatalf("failed to connect to Docker: %v", err)
	}

	// Créer le réseau Docker si nécessaire
	if err := photonDocker.EnsureNetwork(ctx); err != nil {
		log.Printf("warning: failed to ensure Docker network: %v", err)
	}

	// Initialiser le client Panel
	remoteClient := remote.NewClient()

	// Initialiser le store API
	store := api.NewMemoryStore()

	// Démarrer l'API HTTP
	go func() {
		addr := fmt.Sprintf("%s:%d", config.Get().API.Host, config.Get().API.Port)
		log.Printf("starting API server on %s", addr)
		if err := api.ListenAndServe(ctx, store, remoteClient); err != nil {
			log.Printf("API server stopped: %v", err)
		}
	}()

	// Attendre le signal d'arrêt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %v, shutting down...", sig)
	cancel()
}
