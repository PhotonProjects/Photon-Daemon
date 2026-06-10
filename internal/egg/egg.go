package egg

// Egg représente un egg au format Pterodactyl natif.
type Egg struct {
	ID          string            `json:"-"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Author      string            `json:"author,omitempty"`
	UUID        string            `json:"uuid,omitempty"`
	DockerImages map[string]string `json:"docker_images"`
	Startup     string            `json:"startup"`
	Environment []EggVariable     `json:"environment,omitempty"`
	Scripts     EggScripts        `json:"scripts"`
	ConfigFiles []ConfigFile      `json:"config_files,omitempty"`
	FeatureLimits FeatureLimits    `json:"feature_limits,omitempty"`
	FileDenylist []string         `json:"file_denylist,omitempty"`
}

// EggVariable définit une variable d'environnement dans un egg.
type EggVariable struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	EnvVariable  string `json:"env_variable"`
	DefaultValue string `json:"default_value"`
	UserViewable bool   `json:"user_viewable"`
	UserEditable bool   `json:"user_editable"`
	Rules        string `json:"rules"`
}

// EggScripts contient les scripts d'installation de l'egg.
type EggScripts struct {
	Installation EggInstallationScript `json:"installation"`
}

// EggInstallationScript définit comment installer le serveur.
type EggInstallationScript struct {
	Script         string `json:"script"`
	ContainerImage string `json:"container_image"`
	Entrypoint     string `json:"entrypoint"`
}

// ConfigParser est le type de parseur pour un fichier de config.
type ConfigParser string

const (
	ConfigParserFile       ConfigParser = "file"
	ConfigParserYaml       ConfigParser = "yaml"
	ConfigParserJson       ConfigParser = "json"
	ConfigParserXml        ConfigParser = "xml"
	ConfigParserIni        ConfigParser = "ini"
	ConfigParserProperties ConfigParser = "properties"
)

// ConfigFile décrit un fichier de configuration à modifier au démarrage.
type ConfigFile struct {
	FileName string          `json:"file"`
	Parser   ConfigParser    `json:"parser"`
	Replace  []ConfigReplace `json:"replace"`
}

// ConfigReplace décrit une substitution dans un fichier de config.
type ConfigReplace struct {
	Match       string       `json:"match"`
	IfValue     string       `json:"if_value,omitempty"`
	ReplaceWith ReplaceValue `json:"replace_with"`
}

// ReplaceValue contient la valeur de remplacement avec son type.
type ReplaceValue struct {
	value     []byte
	valueType ValueType
}

func (rv ReplaceValue) Value() []byte { return rv.value }

func (rv ReplaceValue) Type() ValueType { return rv.valueType }

func (rv ReplaceValue) String() string { return string(rv.value) }

// ValueType représente le type JSON d'une valeur.
type ValueType int

const (
	ValueString  ValueType = iota
	ValueNumber
	ValueBoolean
	ValueNull
)

// FeatureLimits définit les limites par défaut de l'egg.
type FeatureLimits struct {
	Memory int `json:"memory"`
	CPU    int `json:"cpu"`
	Disk   int `json:"disk"`
}

// InstallationScript est renvoyé par le Panel au moment de l'install.
type InstallationScript struct {
	ContainerImage string `json:"container_image"`
	Entrypoint     string `json:"entrypoint"`
	Script         string `json:"script"`
}

// ResolvedEgg est le résultat de la résolution d'un egg avec les valeurs utilisateur.
type ResolvedEgg struct {
	ResolvedStartup string
	DockerImage     string
	Env             map[string]string
	ResolvedConfigs []ResolvedConfigFile
	InstallScript   InstallationScript
}

// ResolvedConfigFile est un fichier de config dont les valeurs sont résolues.
type ResolvedConfigFile struct {
	FileName string
	Parser   ConfigParser
	Replace  []ConfigReplace
}
