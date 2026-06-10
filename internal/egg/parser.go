package egg

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var defaultLimits = FeatureLimits{
	Memory: 1024,
	CPU:    0,
	Disk:   0,
}

// ParseEgg parse un JSON brut en Egg validé.
func ParseEgg(data []byte) (*Egg, error) {
	var e Egg
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("egg: invalid JSON: %w", err)
	}
	e.Normalize()
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return &e, nil
}

// ParseEggFromReader parse un flux JSON en Egg validé.
func ParseEggFromReader(r io.Reader) (*Egg, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("egg: failed to read data: %w", err)
	}
	return ParseEgg(data)
}

// Normalize applique les valeurs par défaut et gère la rétrocompatibilité.
func (e *Egg) Normalize() {
	// Normaliser les variables d'environnement
	for i := range e.Environment {
		v := &e.Environment[i]
		if v.EnvVariable == "" {
			v.EnvVariable = strings.ToUpper(v.Name)
		}
		// Remplacer les espaces par des underscores dans le nom de variable
		v.EnvVariable = strings.NewReplacer(" ", "_", "-", "_").Replace(v.EnvVariable)
	}

	// Appliquer les feature limits par défaut si non définies
	if e.FeatureLimits == (FeatureLimits{}) {
		e.FeatureLimits = defaultLimits
	}
	if e.FeatureLimits.Memory == 0 {
		e.FeatureLimits.Memory = defaultLimits.Memory
	}

	// S'assurer que les scripts existent
	if e.Scripts.Installation.Entrypoint == "" {
		e.Scripts.Installation.Entrypoint = "/bin/bash"
	}
}

// Validate vérifie l'intégrité de l'egg.
func (e *Egg) Validate() error {
	var errs []string

	if e.Name == "" {
		errs = append(errs, "name is required")
	}
	if len(e.DockerImages) == 0 {
		errs = append(errs, "at least one docker_image is required")
	}
	if e.Startup == "" {
		errs = append(errs, "startup command is required")
	}
	if e.Scripts.Installation.Script == "" {
		errs = append(errs, "installation script is required")
	}
	if e.Scripts.Installation.ContainerImage == "" {
		errs = append(errs, "installation container_image is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("egg: validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
