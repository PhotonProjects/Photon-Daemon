package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Debug bool `yaml:"debug"`

	App  AppConfig  `yaml:"app"`
	API  APIConfig  `yaml:"api"`
	Panel PanelConfig `yaml:"panel"`
	System SystemConfig `yaml:"system"`
	Docker DockerConfig `yaml:"docker"`

	Throttles      ThrottlesConfig      `yaml:"throttles"`
	CrashDetection CrashDetectionConfig `yaml:"crash_detection"`
}

type AppConfig struct {
	Name              string `yaml:"name"`
	TmpfsSize         uint   `yaml:"tmpfs_size"`
	ContainerPIDLimit int64  `yaml:"container_pid_limit"`
}

type APIConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	SSL  SSLConfig `yaml:"ssl"`
}

type SSLConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type PanelConfig struct {
	BaseURL   string `yaml:"base_url"`
	AuthToken string `yaml:"auth_token"`
}

type SystemConfig struct {
	DataDir                string `yaml:"data_dir"`
	TmpDir                 string `yaml:"tmp_dir"`
	LogDir                 string `yaml:"log_dir"`
	Timezone               string `yaml:"timezone"`
	CheckPermissionsOnBoot bool   `yaml:"check_permissions_on_boot"`
}

type DockerConfig struct {
	Network        DockerNetworkConfig        `yaml:"network"`
	InstallerLimits DockerInstallerLimits     `yaml:"installer_limits"`
	Registries     map[string]RegistryAuth    `yaml:"registries,omitempty"`
}

type DockerNetworkConfig struct {
	Name      string   `yaml:"name"`
	Mode      string   `yaml:"mode"`
	Interface string   `yaml:"interface"`
	DNS       []string `yaml:"dns,omitempty"`
}

type DockerInstallerLimits struct {
	Memory int64 `yaml:"memory"`
	CPU    int64 `yaml:"cpu"`
}

type RegistryAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ThrottlesConfig struct {
	Enabled            bool `yaml:"enabled"`
	Lines              int  `yaml:"lines"`
	MaxTriggerCount    int  `yaml:"maximum_trigger_count"`
	LineResetInterval  int  `yaml:"line_reset_interval"`
	DecayInterval      int  `yaml:"decay_interval"`
	StopGracePeriod    int  `yaml:"stop_grace_period"`
}

type CrashDetectionConfig struct {
	Enabled                  bool `yaml:"enabled"`
	Timeout                  int  `yaml:"timeout"`
	DetectCleanExitAsCrash   bool `yaml:"detect_clean_exit_as_crash"`
}

var global Config

func Get() *Config {
	return &global
}

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: failed to read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &global); err != nil {
		return fmt.Errorf("config: failed to parse %s: %w", path, err)
	}
	setDefaults()
	return nil
}

func setDefaults() {
	if global.App.Name == "" {
		global.App.Name = "Photon Daemon"
	}
	if global.App.TmpfsSize == 0 {
		global.App.TmpfsSize = 100
	}
	if global.App.ContainerPIDLimit == 0 {
		global.App.ContainerPIDLimit = 512
	}
	if global.API.Host == "" {
		global.API.Host = "0.0.0.0"
	}
	if global.API.Port == 0 {
		global.API.Port = 8080
	}
	if global.System.DataDir == "" {
		global.System.DataDir = "/var/lib/photon"
	}
	if global.System.TmpDir == "" {
		global.System.TmpDir = "/tmp/photon"
	}
	if global.System.LogDir == "" {
		global.System.LogDir = "/var/log/photon"
	}
	if global.System.Timezone == "" {
		global.System.Timezone = "UTC"
	}
	if global.Docker.Network.Name == "" {
		global.Docker.Network.Name = "photon_nw"
	}
	if global.Docker.Network.Mode == "" {
		global.Docker.Network.Mode = "bridge"
	}
	if global.Docker.InstallerLimits.Memory == 0 {
		global.Docker.InstallerLimits.Memory = 1024
	}
	if global.Docker.InstallerLimits.CPU == 0 {
		global.Docker.InstallerLimits.CPU = 100
	}
	if global.Throttles.Lines == 0 {
		global.Throttles.Lines = 2000
	}
	if global.Throttles.MaxTriggerCount == 0 {
		global.Throttles.MaxTriggerCount = 5
	}
	if global.Throttles.LineResetInterval == 0 {
		global.Throttles.LineResetInterval = 100
	}
	if global.Throttles.DecayInterval == 0 {
		global.Throttles.DecayInterval = 10000
	}
	if global.Throttles.StopGracePeriod == 0 {
		global.Throttles.StopGracePeriod = 15
	}
	if !global.CrashDetection.Enabled {
		global.CrashDetection.Enabled = true
	}
	if global.CrashDetection.Timeout == 0 {
		global.CrashDetection.Timeout = 60
	}
}
