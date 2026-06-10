# Photon Daemon — Architecture

## Sommaire
1. [Configuration Daemon](#1-configuration-daemon)
2. [Modèle de données — Egg](#2-modèle-de-données--egg)
3. [Configuration serveur (envoyée par le Panel)](#3-configuration-serveur-envoyée-par-le-panel)
4. [Contrat API — Panel → Daemon](#4-contrat-api--panel--daemon)
5. [Contrat API — Daemon → Panel](#5-contrat-api--daemon--panel)
6. [Cycle de vie d'un serveur](#6-cycle-de-vie-dun-serveur)
7. [Structure du projet Go](#7-structure-du-projet-go)

---

## 1. Configuration Daemon

Fichier : `/etc/photon/config.yml`

```yaml
debug: false

app:
  name: "Photon Daemon"
  tmpfs_size: 100          # MB, taille du /tmp dans les containers
  container_pid_limit: 512

api:                       # API que le Panel appelle
  host: "0.0.0.0"
  port: 8080
  ssl:
    enabled: false
    cert_file: ""
    key_file: ""

panel:                     # Connexion vers le Panel
  base_url: "https://panel.example.com"
  auth_token: "pt_..."

system:
  data_dir: "/var/lib/photon"      # Données des serveurs
  tmp_dir: "/tmp/photon"           # Temp d'installation
  log_dir: "/var/log/photon"       # Logs d'installation
  timezone: "UTC"
  check_permissions_on_boot: true

docker:
  network:
    name: "photon_nw"
    mode: "bridge"
    interface: "172.19.0.1"
    dns:
      - "1.1.1.1"
      - "8.8.8.8"
  installer_limits:        # Limites du container d'install
    memory: 1024
    cpu: 100
  registries:               # Auth pour registry privés
    ghcr.io:
      username: "user"
      password: "token"

throttles:                  # Anti-spam console
  enabled: true
  lines: 2000
  max_trigger_count: 5
  line_reset_interval: 100  # ms
  decay_interval: 10000     # ms
  stop_grace_period: 15

crash_detection:
  enabled: true
  timeout: 60               # secondes entre crashes
  detect_clean_exit_as_crash: true
```

---

## 2. Modèle de données — Egg

### Egg (format Pterodactyl natif)

```json
{
  "name": "Minecraft Java",
  "description": "Serveur Minecraft vanilla/Paper/Forge",
  "author": "Photon",
  "docker_images": {
    "ghcr.io/photon/games:minecraft-java": "Minecraft Java"
  },
  "startup": "java -Xms{{SERVER_MEMORY}}M -jar {{SERVER_JARFILE}}",
  "environment": [
    {
      "name": "Server JAR File",
      "description": "Nom du fichier JAR",
      "env_variable": "SERVER_JARFILE",
      "default_value": "server.jar",
      "user_viewable": true,
      "user_editable": true,
      "rules": "required|string|max:50"
    }
  ],
  "scripts": {
    "installation": {
      "script": "apt-get update && apt-get install -y curl ...",
      "container_image": "debian:bullseye",
      "entrypoint": "/bin/bash"
    }
  },
  "config_files": [
    {
      "file": "server.properties",
      "parser": "properties",
      "replace": [
        {
          "match": "server-port",
          "replace_with": "{{SERVER_PORT}}"
        }
      ]
    }
  ],
  "feature_limits": {
    "memory": 1024,
    "cpu": 0,
    "disk": 0
  },
  "file_denylist": [
    "*.exe", "*.bat"
  ]
}
```

### Règles de validation des variables

| Règle | Description |
|-------|-------------|
| `required` | Champ obligatoire |
| `string` | Doit être une chaîne |
| `numeric` | Doit être numérique |
| `integer` | Doit être un entier |
| `boolean` | Doit être un booléen |
| `max:N` | Maximum N caractères/valeur |
| `min:N` | Minimum N caractères/valeur |
| `regex:...` | Doit matcher l'expression |

---

## 3. Configuration serveur (envoyée par le Panel)

Payload envoyé par le Panel au Daemon lors de la création/sync :

```json
{
  "uuid": "uuid-du-serveur",
  "settings": {
    "name": "Mon Serveur",
    "suspended": false,
    "skip_egg_scripts": false,
    "invocation": "java -Xms2048M -jar paper.jar",
    "environment_variables": {
      "SERVER_JARFILE": "paper.jar",
      "SERVER_MEMORY": "2048"
    },
    "build": {
      "memory_limit": 4096,
      "swap": 0,
      "cpu_limit": 200,
      "io_weight": 500,
      "disk_limit": 10240,
      "threads": null,
      "oom_disabled": true
    },
    "allocations": {
      "default": {
        "ip": "0.0.0.0",
        "port": 25565
      },
      "additional": []
    },
    "mounts": [],
    "egg": {
      "docker_images": {
        "ghcr.io/photon/games:minecraft-java": "Minecraft Java"
      },
      "scripts": {
        "installation": {
          "script": "...",
          "container_image": "debian:bullseye",
          "entrypoint": "/bin/bash"
        }
      }
    }
  },
  "process_configuration": {
    "startup": {
      "done": ["Done!", "For help"],
      "user_interaction": [],
      "strip_ansi": true
    },
    "stop": {
      "type": "command",
      "value": "stop"
    },
    "configs": [
      {
        "file": "server.properties",
        "parser": "properties",
        "replace": [
          {
            "match": "server-port",
            "replace_with": "25565"
          }
        ]
      }
    ]
  }
}
```

---

## 4. Contrat API — Panel → Daemon

Le Daemon expose une API REST + WebSocket que le Panel appelle.

### REST

| Méthode | Path | Body | Description |
|---------|------|------|-------------|
| `POST` | `/api/servers` | `ServerConfiguration` | Crée un serveur |
| `GET` | `/api/servers/:uuid` | — | Détail d'un serveur |
| `DELETE` | `/api/servers/:uuid` | — | Supprime un serveur |
| `POST` | `/api/servers/:uuid/power` | `{ action: "start" \| "stop" \| "restart" \| "kill" }` | Contrôle power |
| `POST` | `/api/servers/:uuid/install` | — | Déclenche (ré)installation |
| `POST` | `/api/servers/:uuid/sync` | — | Force la sync Panel → Daemon |
| `GET` | `/api/servers/:uuid/logs` | — | Récupère les logs |
| `GET` | `/api/servers/:uuid/files/*` | — | Listing fichier (via SFTP ou API) |
| `GET` | `/api/servers` | — | Liste tous les serveurs du Daemon |

### WebSocket

```
WS /api/servers/:uuid/ws?token=<jwt>
```

Événements Panel → Daemon :
```json
{"event": "power", "args": ["start"]}
{"event": "command", "args": ["say hello"]}
{"event": "stats"}
```

Événements Daemon → Panel :
```json
{"event": "status", "args": ["running"]}
{"event": "console output", "args": ["[INFO] Server started"]}
{"event": "stats", "args": [{"memory": 1024, "cpu": 45, "disk": 2048}]}
{"event": "install output", "args": ["Downloading..."]}
{"event": "install started", "args": []}
{"event": "install completed", "args": []}
{"event": "daemon message", "args": ["Démarrage..."]}
```

---

## 5. Contrat API — Daemon → Panel

Le Daemon appelle le Panel pour récupérer des infos et synchroniser l'état.

| Méthode | Path | Body | Description |
|---------|------|------|-------------|
| `GET` | `/api/servers/:uuid/config` | — | Récupère la config serveur |
| `GET` | `/api/servers/:uuid/install` | — | Récupère le script d'install |
| `POST` | `/api/servers/:uuid/install/status` | `{ successful, reinstall }` | Notifie la fin d'install |
| `POST` | `/api/servers/:uuid/activity` | `{ event, metadata }` | Log d'activité |
| `POST` | `/api/servers/:uuid/transfer` | — | Statut de transfert |

---

## 6. Cycle de vie d'un serveur

```
                       ┌──────────────┐
                       │   PENDING    │
                       └──────┬───────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   INSTALLING    │ ← Container d'install actif
                    └────────┬────────┘
                             │ succès
                             ▼
                    ┌─────────────────┐
                    │   INSTALLED     │ ← Prêt à démarrer
                    └────────┬────────┘
                             │ start
                             ▼
                    ┌─────────────────┐
               ┌───│    STARTING     │───┐
               │   └────────┬────────┘   │
               │            │            │
               │            ▼            │
               │   ┌─────────────────┐   │
               │   │    RUNNING      │   │
               │   └────────┬────────┘   │
               │            │            │
               │     stop / crash        │
               │            │            │
               │            ▼            │
               │   ┌─────────────────┐   │
               └───│    STOPPING     │◄──┘
                   └────────┬────────┘
                            │
                            ▼
                   ┌─────────────────┐
                   │    OFFLINE      │
                   └─────────────────┘
```

États :
- `pending` — Serveur créé, pas encore installé
- `installing` — Installation en cours
- `installed` — Installation réussie
- `starting` — Démarrage
- `running` — En cours d'exécution
- `stopping` — Arrêt en cours
- `offline` — Arrêté

---

## 7. Structure du projet Go

```
photon-daemon/
├── cmd/
│   └── photon-daemon/
│       └── main.go              # Entrypoint
├── config/
│   ├── config.go                # Struct & chargement config.yml
│   ├── registry.go              # Auth registries Docker
│   └── config_test.go
├── internal/
│   ├── egg/
│   │   ├── egg.go               # Struct Egg (Pterodactyl format)
│   │   ├── parser.go            # Parse JSON → Egg
│   │   ├── validator.go         # Valide les règles des variables
│   │   ├── resolver.go          # Substitution des variables
│   │   └── egg_test.go
│   ├── docker/
│   │   ├── client.go            # Wrapper Docker SDK
│   │   ├── image.go             # Pull, cache, prune
│   │   ├── network.go           # Gestion du network
│   │   └── container.go         # Create, start, stop, remove
│   ├── server/
│   │   ├── server.go            # Struct Server (état, config)
│   │   ├── lifecycle.go         # Machine à états
│   │   ├── install.go           # Processus d'installation
│   │   ├── power.go             # Start/stop/restart/kill
│   │   ├── config_parser.go     # Parsing des config files
│   │   ├── crash.go             # Détection & auto-restart
│   │   ├── events.go            # Émetteur d'événements
│   │   ├── resources.go         # Stats CPU/RAM/Disk
│   │   └── server_test.go
│   ├── filesystem/
│   │   ├── filesystem.go        # Opérations fichier
│   │   ├── permissions.go       # Réglage des permissions
│   │   └── filesystem_test.go
│   └── remote/
│       ├── client.go            # Client HTTP vers Panel
│       ├── types.go             # Types partagés Panel↔Daemon
│       └── client_test.go
├── api/
│   ├── router.go                # Routes HTTP
│   ├── middleware.go            # Auth, logging
│   ├── handlers.go              # Handlers REST
│   ├── websocket.go             # WebSocket handler
│   └── api_test.go
├── events/
│   └── bus.go                   # Event bus pub/sub
├── throttles/
│   └── throttles.go             # Anti-spam console
├── go.mod
└── go.sum
```

### Dépendances clés

| Package | Usage |
|---------|-------|
| `github.com/docker/docker` | SDK Docker |
| `github.com/gorilla/websocket` | WebSocket Panel |
| `github.com/gorilla/mux` ou `chi` | Routeur HTTP |
| `gopkg.in/yaml.v3` | Parse config.yml |
| `github.com/apex/log` | Logging structuré |
| `github.com/beevik/etree` | Parse XML config files |
| `github.com/magiconair/properties` | Parse .properties |
| `gopkg.in/ini.v1` | Parse .ini |
| `gopkg.in/yaml.v3` | Parse YAML config files |
| `github.com/buger/jsonparser` | Manipulation JSON |

---

---

## 8. Egg Parser — Design détaillé

### 8.1 Types du package `internal/egg/`

#### Egg (racine)

```go
type Egg struct {
    ID          string
    Name        string
    Description string
    Author      string
    UUID        string

    // Map d'images Docker disponibles (image → display name)
    DockerImages map[string]string

    // Commande startup avec placeholders {{VAR}}
    Startup string

    // Variables d'environnement définies par l'egg
    Environment []EggVariable

    // Script d'installation
    Scripts EggScripts

    // Fichiers de configuration à parser au démarrage
    ConfigFiles []ConfigFile

    // Feature limits par défaut
    FeatureLimits FeatureLimits

    // Fichiers/bloqués
    FileDenylist []string
}
```

#### EggVariable

```go
type EggVariable struct {
    Name         string // Display name
    Description  string
    EnvVariable  string // Nom de la variable d'env (ex: SERVER_JARFILE)
    DefaultValue string
    UserViewable bool
    UserEditable bool
    Rules        string // Chaîne de règles (ex: "required|string|max:50")
}
```

#### EggScripts

```go
type EggScripts struct {
    Installation EggInstallationScript
}
```

#### EggInstallationScript

```go
type EggInstallationScript struct {
    Script         string // Bash script
    ContainerImage string // Image Docker pour l'install (ex: debian:bullseye)
    Entrypoint     string // Entrypoint (ex: /bin/bash)
}
```

#### ConfigFile

```go
type ConfigFile struct {
    FileName string         // Chemin relatif (ex: server.properties)
    Parser   ConfigParser   // "file" | "yaml" | "json" | "xml" | "ini" | "properties"
    Replace  []ConfigReplace
}

type ConfigParser string

const (
    ConfigParserFile       ConfigParser = "file"
    ConfigParserYaml       ConfigParser = "yaml"
    ConfigParserJson       ConfigParser = "json"
    ConfigParserXml        ConfigParser = "xml"
    ConfigParserIni        ConfigParser = "ini"
    ConfigParserProperties ConfigParser = "properties"
)

type ConfigReplace struct {
    Match       string       // Clé à trouver
    IfValue     string       // Optionnel : ne remplacer que si valeur actuelle == IfValue
    ReplaceWith ReplaceValue // Valeur de remplacement (résolue depuis variable)
}

type ReplaceValue struct {
    value     []byte
    valueType ValueType // string | number | boolean | null
}
```

#### FeatureLimits

```go
type FeatureLimits struct {
    Memory int // MB, 0 = illimité
    CPU    int // 0 = illimité
    Disk   int // MB, 0 = illimité
}
```

#### InstallationScript (envoyé par le Panel au moment de l'install)

```go
type InstallationScript struct {
    ContainerImage string `json:"container_image"`
    Entrypoint     string `json:"entrypoint"`
    Script         string `json:"script"`
}
```

### 8.2 Parseur — Flux

```
JSON brut (Panel)
    │
    ▼
┌──────────────────┐
│   json.Unmarshal │  → Egg struct brute
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│   Normalize()    │  → Applique les valeurs par défaut
└──────┬───────────┘       Convertit les champs legacy
       │
       ▼
┌──────────────────┐
│   Validate()     │  → Vérifie l'intégrité de l'egg
└──────┬───────────┘       (name requis, au moins 1 docker_image, etc.)
       │
       ▼
┌──────────────────┐
│   Egg prêt       │  → Utilisé par le server
└──────────────────┘
```

#### Interface du parseur

```go
// ParseEgg parse un JSON brut en Egg validé
func ParseEgg(data []byte) (*Egg, error)

// ParseEggFromReader parse un flux JSON en Egg validé
func ParseEggFromReader(r io.Reader) (*Egg, error)
```

#### Normalisation

```go
// Normalize applique les valeurs par défaut et gère la rétrocompatibilité
func (e *Egg) Normalize() {
    for i := range e.Environment {
        if e.Environment[i].EnvVariable == "" {
            e.Environment[i].EnvVariable = strings.ToUpper(e.Environment[i].Name)
        }
    }
    if e.FeatureLimits.Memory == 0 {
        e.FeatureLimits.Memory = 1024
    }
}
```

#### Validation de base

```go
func (e *Egg) Validate() error {
    if e.Name == "" {
        return errors.New("egg: name is required")
    }
    if len(e.DockerImages) == 0 {
        return errors.New("egg: at least one docker image is required")
    }
    if e.Startup == "" {
        return errors.New("egg: startup command is required")
    }
    return nil
}
```

### 8.3 Validateur de variables

Système de règles inspiré des règles de validation Pterodactyl.

#### Architecture

```
Rules string (ex: "required|string|max:50")
    │
    ▼
┌──────────────────┐
│   ParseRules()   │  → Découpe "required|string|max:50"
└──────┬───────────┘    → Retourne []Rule
       │
       ▼
┌──────────────────┐
│   Validate()     │  → Applique chaque règle à la valeur
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│   []ValidationError│
└──────────────────┘
```

#### Types

```go
type Rule interface {
    Name() string
    Validate(value string) error
}

// Règles concrètes
type RuleRequired struct{}
type RuleString struct{}
type RuleNumeric struct{}
type RuleInteger struct{}
type RuleBoolean struct{}
type RuleMax struct{ Max int }
type RuleMin struct{ Min int }
type RuleRegex struct{ Pattern *regexp.Regexp }
```

#### Parseur de règles

```go
// ParseRules convertit "required|string|max:50" en []Rule
func ParseRules(rules string) ([]Rule, error)
```

Implémentation :

| Entrée | Règles produites |
|--------|------------------|
| `required\|string` | RuleRequired, RuleString |
| `numeric\|min:1\|max:65535` | RuleNumeric, RuleMin{1}, RuleMax{65535} |
| `required\|regex:^[a-z]+$` | RuleRequired, RuleRegex{`^[a-z]+$`} |
| `boolean` | RuleBoolean |

#### Validateur

```go
// ValidateVar applique les règles à une valeur
func ValidateVar(variable EggVariable, value string) error {
    rules, err := ParseRules(variable.Rules)
    if err != nil {
        return err
    }

    for _, rule := range rules {
        if err := rule.Validate(value); err != nil {
            return fmt.Errorf("%s: %s: %w", variable.EnvVariable, rule.Name(), err)
        }
    }
    return nil
}
```

### 8.4 Resolver — Substitution des variables

Le resolver prend un egg + un map de valeurs utilisateur et produit la startup command finale et les config files résolus.

#### Types

```go
type ResolvedEgg struct {
    // Commande startup avec variables substituées
    // Ex: "java -Xms2048M -jar paper.jar"
    ResolvedStartup string

    // Docker image sélectionnée
    DockerImage string

    // Variables d'env finales (clé → valeur)
    Env map[string]string

    // Fichiers de config résolus (avec valeurs substituées)
    ResolvedConfigs []ResolvedConfigFile

    // Script d'install
    InstallScript InstallationScript
}

type ResolvedConfigFile struct {
    FileName string
    Parser   ConfigParser
    Replace  []ConfigReplace // Valeurs déjà substituées
}
```

#### Interface

```go
// ResolveEgg prend un Egg + valeurs utilisateur et résout toutes les variables
func ResolveEgg(egg *Egg, userVars map[string]string, selectedImage string) (*ResolvedEgg, error)
```

#### Flux de résolution

```
Egg.Startup = "java -Xms{{SERVER_MEMORY}}M -jar {{SERVER_JARFILE}}"
userVars = { "SERVER_MEMORY": "2048", "SERVER_JARFILE": "paper.jar" }
    │
    ▼
┌──────────────────────────┐
│ 1. Valider les variables │  → ValidateVar() sur chaque (variable, valeur)
└──────────────────────────┘
    │
    ▼
┌──────────────────────────┐
│ 2. Substitution startup  │  → {{SERVER_MEMORY}} → 2048
└──────────────────────────┘      {{SERVER_JARFILE}} → paper.jar
    │
    ▼
┌──────────────────────────┐
│ 3. Substitution config   │  → {{SERVER_PORT}} → 25565
└──────────────────────────┘    dans chaque ConfigFile.Replace
    │
    ▼
┌──────────────────────────┐
│ 4. Build env map         │  → { "SERVER_MEMORY": "2048", ... }
└──────────────────────────┘
    │
    ▼
┌──────────────────────────┐
│ 5. Sélection image       │  → selectedImage ou première de DockerImages
└──────────────────────────┘
```

#### Substitution

```go
// ResolveStartup remplace les {{VAR}} dans la commande startup
func ResolveStartup(template string, vars map[string]string) string

// ResolveConfigValue remplace {{VAR}} dans une valeur de config
func ResolveConfigValue(template string, vars map[string]string) string
```

Format des placeholders : `{{VARIABLE_NAME}}`

### 8.5 Flux complet Egg → Container

```
Panel envoie :
  - Configuration serveur (avec invocation déjà résolue)
  - Egg complet (variables, scripts, configs)
  - Valeurs utilisateur
    │
    ▼
┌─────────────────────────────────────┐
│ egg.ParseEgg(json)                  │  → Struct Egg
│ egg.Validate()                      │  → Intégrité
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│ validator.ParseRules(variable.Rules)│  → []Rule
│ validator.ValidateVar(var, value)   │  → Validation
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│ resolver.ResolveEgg(egg, userVars)  │  → ResolvedEgg
│   - Substitution startup            │
│   - Substitution config files       │
│   - Build env map                   │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│ Install phase                       │
│   docker.ImagePull(container_image) │
│   docker.CreateContainer(install)   │
│   docker.StartContainer             │
│   WaitForCompletion                 │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│ Run phase                           │
│   docker.CreateContainer(run)       │
│     Image = selected docker_image   │
│     Cmd  = resolved startup         │
│     Env  = env map                  │
│     Mounts = server data directory  │
│   docker.StartContainer             │
│   config_parser.UpdateConfigs()     │
└─────────────────────────────────────┘
```

### 8.6 Tests

Le package `internal/egg/` doit tester :

| Test | Description |
|------|-------------|
| `TestParseEgg` | Parse un egg JSON valide |
| `TestParseEgg_InvalidJSON` | JSON invalide → erreur |
| `TestParseEgg_MissingRequired` | Egg sans name → erreur |
| `TestNormalizeEgg` | Valeurs par défaut appliquées |
| `TestParseRules` | Chaîne de règles → []Rule |
| `TestParseRules_InvalidRule` | Règle inconnue → erreur |
| `TestValidateVar_Valid` | Valeur valide → nil |
| `TestValidateVar_Invalid` | Valeur invalide → erreur |
| `TestResolveStartup` | Substitution {{VAR}} |
| `TestResolveStartup_MissingVar` | Variable manquante → laissée telle quelle |
| `TestResolveEgg` | Egg complet résolu |
| `TestResolveConfigValue` | Substitution dans les config files |

### 8.7 Exemple complet

```go
// Données reçues du Panel
eggJSON := `{
  "name": "Minecraft Java",
  "docker_images": {"ghcr.io/photon/games:mc": "Minecraft"},
  "startup": "java -Xms{{SERVER_MEMORY}}M -jar {{SERVER_JARFILE}}",
  "environment": [
    {
      "name": "Server JAR File",
      "env_variable": "SERVER_JARFILE",
      "default_value": "server.jar",
      "rules": "required|string|max:50"
    }
  ],
  "scripts": {
    "installation": {
      "script": "curl -o /mnt/server/server.jar https://...",
      "container_image": "debian:bullseye",
      "entrypoint": "/bin/bash"
    }
  }
}`

userVars := map[string]string{
    "SERVER_JARFILE": "paper-1.20.jar",
}

// 1. Parse
egg, _ := egg.ParseEgg([]byte(eggJSON))

// 2. Validate
egg.Validate()

// 3. Resolve
resolved, _ := egg.ResolveEgg(egg, userVars, "")

// Résultat :
// resolved.ResolvedStartup = "java -Xms1024M -jar paper-1.20.jar"
// resolved.DockerImage     = "ghcr.io/photon/games:mc"
// resolved.Env             = {"SERVER_JARFILE": "paper-1.20.jar", "SERVER_MEMORY": "1024"}
// resolved.InstallScript   = { Script: "curl ...", ContainerImage: "debian:bullseye", Entrypoint: "/bin/bash" }
```

## Commandes make

```makefile
build:       go build -o bin/photon-daemon cmd/photon-daemon/main.go
run:         go run cmd/photon-daemon/main.go
test:        go test ./...
lint:        golangci-lint run
docker:      docker build -t photon-daemon .
```
