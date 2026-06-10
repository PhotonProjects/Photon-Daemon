package egg

import (
	"fmt"
	"regexp"
	"strings"
)

var placeholderRe = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// ResolveStartup remplace les {{VAR}} dans une commande startup.
// Les variables non trouvées sont laissées telles quelles.
func ResolveStartup(template string, vars map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		name := strings.TrimRight(strings.TrimLeft(match, "{{"), "}")
		if val, ok := vars[name]; ok {
			return val
		}
		return match
	})
}

// ResolveConfigValue remplace les {{VAR}} dans une valeur de config.
func ResolveConfigValue(template string, vars map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		name := strings.TrimRight(strings.TrimLeft(match, "{{"), "}")
		if val, ok := vars[name]; ok {
			return val
		}
		return match
	})
}

// ResolveEgg prend un Egg + valeurs utilisateur et résout toutes les variables.
func ResolveEgg(egg *Egg, userVars map[string]string, selectedImage string) (*ResolvedEgg, error) {
	// Construire la map de toutes les variables avec leurs valeurs
	env := make(map[string]string, len(egg.Environment))

	// Valeurs système (calculées)
	env["SERVER_MEMORY"] = fmt.Sprintf("%d", egg.FeatureLimits.Memory)

	// Appliquer les variables de l'egg : défaut → override utilisateur
	for _, v := range egg.Environment {
		val := v.DefaultValue
		if uv, ok := userVars[v.EnvVariable]; ok && uv != "" {
			val = uv
		}
		env[v.EnvVariable] = val
	}

	// Override direct par userVars pour les variables système
	for k, v := range userVars {
		env[k] = v
	}

	// Valider chaque variable qui a des règles
	for _, v := range egg.Environment {
		if v.Rules != "" {
			if err := ValidateVar(v, env[v.EnvVariable]); err != nil {
				return nil, fmt.Errorf("resolve: %w", err)
			}
		}
	}

	// Résoudre la startup
	resolvedStartup := ResolveStartup(egg.Startup, env)

	// Sélectionner l'image Docker
	image := selectedImage
	if image == "" {
		for img := range egg.DockerImages {
			image = img
			break
		}
	}

	// Résoudre les fichiers de config
	resolvedConfigs := make([]ResolvedConfigFile, 0, len(egg.ConfigFiles))
	for _, cf := range egg.ConfigFiles {
		resolvedReplaces := make([]ConfigReplace, 0, len(cf.Replace))
		for _, r := range cf.Replace {
			replacedVal := ResolveConfigValue(r.ReplaceWith.String(), env)
			resolvedReplaces = append(resolvedReplaces, ConfigReplace{
				Match:   r.Match,
				IfValue: r.IfValue,
				ReplaceWith: ReplaceValue{
					value:     []byte(replacedVal),
					valueType: ValueString,
				},
			})
		}
		resolvedConfigs = append(resolvedConfigs, ResolvedConfigFile{
			FileName: cf.FileName,
			Parser:   cf.Parser,
			Replace:  resolvedReplaces,
		})
	}

	return &ResolvedEgg{
		ResolvedStartup: resolvedStartup,
		DockerImage:     image,
		Env:             env,
		ResolvedConfigs: resolvedConfigs,
		InstallScript: InstallationScript{
			ContainerImage: egg.Scripts.Installation.ContainerImage,
			Entrypoint:     egg.Scripts.Installation.Entrypoint,
			Script:         egg.Scripts.Installation.Script,
		},
	}, nil
}

// NewReplaceValue crée une ReplaceValue depuis une string simple.
func NewReplaceValue(s string) ReplaceValue {
	return ReplaceValue{value: []byte(s), valueType: ValueString}
}
