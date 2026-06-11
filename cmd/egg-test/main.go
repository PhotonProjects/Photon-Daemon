package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Maj-Studios/Photon-Daemon/internal/egg"
)

func main() {
	eggFile := flag.String("egg", "", "chemin vers le fichier JSON de l'egg (ex: egg.json)")
	varsFile := flag.String("vars", "", "chemin vers le fichier JSON des variables utilisateur (optionnel)")
	image := flag.String("image", "", "sélectionner une image Docker spécifique (optionnel)")
	flag.Parse()

	if *eggFile == "" {
		fmt.Println("Usage: egg-test -egg <file.json> [-vars <vars.json>] [-image <docker_image>]")
		fmt.Println("  -egg    fichier JSON de l'egg (format Pterodactyl)")
		fmt.Println("  -vars   fichier JSON avec les variables utilisateur { \"VAR\": \"val\" }")
		fmt.Println("  -image  sélectionner une image Docker spécifique")
		fmt.Println("\nSi -vars n'est pas fourni, les valeurs par défaut de l'egg sont utilisées.")
		os.Exit(1)
	}

	// Lire l'egg
	eggData, err := os.ReadFile(*eggFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: impossible de lire %s: %v\n", *eggFile, err)
		os.Exit(1)
	}

	// Parser l'egg
	fmt.Println("=== Parse Egg ===")
	e, err := egg.ParseEgg(eggData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur de parsing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Name:          %s\n", e.Name)
	fmt.Printf("✓ Description:   %s\n", e.Description)
	fmt.Printf("✓ Author:        %s\n", e.Author)
	fmt.Printf("✓ Startup:       %s\n", e.Startup)
	fmt.Printf("✓ Docker Images: %d\n", len(e.DockerImages))
	for img, label := range e.DockerImages {
		fmt.Printf("    - %s  (%s)\n", img, label)
	}
	fmt.Printf("✓ Variables:     %d\n", len(e.Environment))
	for _, v := range e.Environment {
		fmt.Printf("    - %-25s défaut: %-15s règles: %s\n", v.EnvVariable, v.DefaultValue, v.Rules)
	}
	fmt.Printf("✓ Config Files:  %d\n", len(e.ConfigFiles))
	for _, cf := range e.ConfigFiles {
		fmt.Printf("    - %-30s parser: %s (%d substitutions)\n", cf.FileName, cf.Parser, len(cf.Replace))
	}
	fmt.Printf("✓ Limits:        mémoire=%dM  cpu=%d  disque=%dM\n",
		e.FeatureLimits.Memory, e.FeatureLimits.CPU, e.FeatureLimits.Disk)
	fmt.Printf("✓ Install image: %s (entrypoint: %s)\n",
		e.Scripts.Installation.ContainerImage, e.Scripts.Installation.Entrypoint)
	fmt.Print("\n")

	// Lire les variables utilisateur
	userVars := make(map[string]string)
	if *varsFile != "" {
		varsData, err := os.ReadFile(*varsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: impossible de lire %s: %v\n", *varsFile, err)
			os.Exit(1)
		}
		if err := json.Unmarshal(varsData, &userVars); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: variables JSON invalides: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Variables utilisateur chargées: %d\n", len(userVars))
		for k, v := range userVars {
			fmt.Printf("    - %s = %s\n", k, v)
		}
		fmt.Print("\n")
	}

	// Résoudre l'egg
	fmt.Println("=== Resolve Egg ===")
	resolved, err := egg.ResolveEgg(e, userVars, *image)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur de résolution: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Docker Image:  %s\n", resolved.DockerImage)
	fmt.Printf("✓ Startup:\n    %s\n", resolved.ResolvedStartup)
	fmt.Printf("✓ Variables d'environnement (%d):\n", len(resolved.Env))
	for k, v := range resolved.Env {
		fmt.Printf("    %s=%s\n", k, v)
	}
	fmt.Printf("✓ Fichiers de config (%d):\n", len(resolved.ResolvedConfigs))
	for _, cf := range resolved.ResolvedConfigs {
		fmt.Printf("    - %s (%s)\n", cf.FileName, cf.Parser)
		for _, r := range cf.Replace {
			fmt.Printf("        match: %-20s → %s\n", r.Match, r.ReplaceWith.String())
		}
	}
	fmt.Printf("✓ Install script:\n    image: %s\n    entrypoint: %s\n    script: %d caractères\n",
		resolved.InstallScript.ContainerImage,
		resolved.InstallScript.Entrypoint,
		len(resolved.InstallScript.Script))
}
