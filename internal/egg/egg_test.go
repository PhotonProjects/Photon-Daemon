package egg

import (
	"io"
	"testing"
)

// ---------- Parser ----------

func TestParseEgg_Valid(t *testing.T) {
	data := `{
		"name": "Minecraft Java",
		"docker_images": {"ghcr.io/photon/games:mc": "Minecraft"},
		"startup": "java -Xms{{SERVER_MEMORY}}M -jar {{SERVER_JARFILE}}",
		"environment": [{
			"name": "Server JAR File",
			"env_variable": "SERVER_JARFILE",
			"default_value": "server.jar",
			"user_viewable": true,
			"user_editable": true,
			"rules": "required|string|max:50"
		}],
		"scripts": {
			"installation": {
				"script": "curl -o /mnt/server/server.jar https://example.com",
				"container_image": "debian:bullseye",
				"entrypoint": "/bin/bash"
			}
		}
	}`

	egg, err := ParseEgg([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if egg.Name != "Minecraft Java" {
		t.Errorf("expected name 'Minecraft Java', got %q", egg.Name)
	}
	if len(egg.DockerImages) != 1 {
		t.Errorf("expected 1 docker image, got %d", len(egg.DockerImages))
	}
	if len(egg.Environment) != 1 {
		t.Errorf("expected 1 env var, got %d", len(egg.Environment))
	}
	if egg.Scripts.Installation.Entrypoint != "/bin/bash" {
		t.Errorf("expected entrypoint /bin/bash, got %q", egg.Scripts.Installation.Entrypoint)
	}
}

func TestParseEgg_InvalidJSON(t *testing.T) {
	data := `{bad json}`
	_, err := ParseEgg([]byte(data))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseEgg_MissingRequired(t *testing.T) {
	data := `{"description": "no name"}`
	_, err := ParseEgg([]byte(data))
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestParseEgg_MissingStartup(t *testing.T) {
	data := `{"name": "test", "docker_images": {"img": "img"}, "scripts": {"installation": {"script": "echo", "container_image": "debian"}}}`
	_, err := ParseEgg([]byte(data))
	if err == nil {
		t.Fatal("expected validation error for missing startup")
	}
}

func TestParseEggFromReader(t *testing.T) {
	data := `{"name": "Test", "docker_images": {"img": "img"}, "startup": "./run.sh", "scripts": {"installation": {"script": "echo hi", "container_image": "alpine"}}}`
	r := &readerMock{data: []byte(data)}
	egg, err := ParseEggFromReader(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if egg.Name != "Test" {
		t.Errorf("expected 'Test', got %q", egg.Name)
	}
}

type readerMock struct {
	data []byte
	off  int
}

func (r *readerMock) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}

// ---------- Normalize ----------

func TestNormalize_EggVariable(t *testing.T) {
	e := &Egg{
		Name:        "test",
		DockerImages: map[string]string{"img": "img"},
		Startup:     "./run.sh",
		Environment: []EggVariable{
			{Name: "Server JAR File", DefaultValue: "server.jar"},
		},
		Scripts: EggScripts{
			Installation: EggInstallationScript{
				Script: "echo", ContainerImage: "debian",
			},
		},
	}
	e.Normalize()
	if e.Environment[0].EnvVariable != "SERVER_JAR_FILE" {
		t.Errorf("expected SERVER_JAR_FILE, got %q", e.Environment[0].EnvVariable)
	}
}

func TestNormalize_DefaultLimits(t *testing.T) {
	e := &Egg{
		Name:         "test",
		DockerImages: map[string]string{"img": "img"},
		Startup:      "./run.sh",
		Scripts: EggScripts{
			Installation: EggInstallationScript{
				Script: "echo", ContainerImage: "debian",
			},
		},
	}
	e.Normalize()
	if e.FeatureLimits.Memory != 1024 {
		t.Errorf("expected default memory 1024, got %d", e.FeatureLimits.Memory)
	}
}

// ---------- Validator ----------

func TestValidate_ValidEgg(t *testing.T) {
	e := &Egg{
		Name:         "test",
		DockerImages: map[string]string{"img": "img"},
		Startup:      "./run.sh",
		Scripts: EggScripts{
			Installation: EggInstallationScript{
				Script: "echo", ContainerImage: "debian",
			},
		},
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_NoName(t *testing.T) {
	e := &Egg{DockerImages: map[string]string{"img": "img"}, Startup: "./run.sh"}
	err := e.Validate()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

// ---------- ParseRules ----------

func TestParseRules_Required(t *testing.T) {
	rules, err := ParseRules("required")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name() != "required" {
		t.Errorf("expected 'required', got %q", rules[0].Name())
	}
}

func TestParseRules_Multiple(t *testing.T) {
	rules, err := ParseRules("required|string|max:50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
}

func TestParseRules_MinMax(t *testing.T) {
	rules, err := ParseRules("min:1|max:65535")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
}

func TestParseRules_Unknown(t *testing.T) {
	_, err := ParseRules("foobar")
	if err == nil {
		t.Fatal("expected error for unknown rule")
	}
}

// ---------- Rule.Validate ----------

func TestRuleRequired_Valid(t *testing.T) {
	r := RuleRequired{}
	if err := r.Validate("hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleRequired_Invalid(t *testing.T) {
	r := RuleRequired{}
	if err := r.Validate(""); err == nil {
		t.Fatal("expected error for empty value")
	}
}

func TestRuleNumeric_Valid(t *testing.T) {
	r := RuleNumeric{}
	if err := r.Validate("42.5"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleNumeric_Invalid(t *testing.T) {
	r := RuleNumeric{}
	if err := r.Validate("notanumber"); err == nil {
		t.Fatal("expected error for non-numeric")
	}
}

func TestRuleInteger_Valid(t *testing.T) {
	r := RuleInteger{}
	if err := r.Validate("42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleInteger_Invalid(t *testing.T) {
	r := RuleInteger{}
	if err := r.Validate("42.5"); err == nil {
		t.Fatal("expected error for non-integer")
	}
}

func TestRuleMax_Valid(t *testing.T) {
	r := RuleMax{Max: 5}
	if err := r.Validate("hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleMax_Invalid(t *testing.T) {
	r := RuleMax{Max: 3}
	if err := r.Validate("hello"); err == nil {
		t.Fatal("expected error for too long value")
	}
}

func TestRuleRegex_Valid(t *testing.T) {
	r, err := buildRule("regex", `^[a-z]+$`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := r.Validate("hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := r.Validate("Hello123"); err == nil {
		t.Fatal("expected error for invalid regex match")
	}
}

// ---------- ValidateVar ----------

func TestValidateVar_Valid(t *testing.T) {
	v := EggVariable{
		EnvVariable: "TEST_VAR",
		Rules:       "required|string|max:50",
	}
	if err := ValidateVar(v, "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVar_Invalid(t *testing.T) {
	v := EggVariable{
		EnvVariable: "TEST_VAR",
		Rules:       "required|max:3",
	}
	if err := ValidateVar(v, "hello"); err == nil {
		t.Fatal("expected validation error")
	}
}

// ---------- Resolver ----------

func TestResolveStartup(t *testing.T) {
	result := ResolveStartup("java -Xms{{MEMORY}}M -jar {{JAR}}", map[string]string{
		"MEMORY": "2048",
		"JAR":    "paper.jar",
	})
	expected := "java -Xms2048M -jar paper.jar"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolveStartup_MissingVar(t *testing.T) {
	result := ResolveStartup("echo {{MISSING}}", map[string]string{})
	if result != "echo {{MISSING}}" {
		t.Errorf("expected placeholder to remain, got %q", result)
	}
}

func TestResolveConfigValue(t *testing.T) {
	result := ResolveConfigValue("{{SERVER_PORT}}", map[string]string{"SERVER_PORT": "25565"})
	if result != "25565" {
		t.Errorf("expected 25565, got %q", result)
	}
}

func TestResolveEgg(t *testing.T) {
	egg := &Egg{
		Name:        "Minecraft",
		DockerImages: map[string]string{"ghcr.io/photon/games:mc": "Minecraft"},
		Startup:     "java -Xms{{SERVER_MEMORY}}M -jar {{JAR}}",
		Environment: []EggVariable{
			{EnvVariable: "JAR", DefaultValue: "server.jar", Rules: "required|string|max:50"},
		},
		FeatureLimits: FeatureLimits{Memory: 2048},
		Scripts: EggScripts{
			Installation: EggInstallationScript{
				Script: "curl ...", ContainerImage: "debian:bullseye", Entrypoint: "/bin/bash",
			},
		},
		ConfigFiles: []ConfigFile{
			{
				FileName: "server.properties",
				Parser:   "properties",
				Replace: []ConfigReplace{
					{Match: "server-port", ReplaceWith: ReplaceValue{value: []byte("{{SERVER_PORT}}"), valueType: ValueString}},
				},
			},
		},
	}

	resolved, err := ResolveEgg(egg, map[string]string{
		"JAR":         "paper.jar",
		"SERVER_PORT": "25565",
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.ResolvedStartup != "java -Xms2048M -jar paper.jar" {
		t.Errorf("unexpected startup: %q", resolved.ResolvedStartup)
	}
	if resolved.DockerImage != "ghcr.io/photon/games:mc" {
		t.Errorf("unexpected image: %q", resolved.DockerImage)
	}
	if resolved.Env["JAR"] != "paper.jar" {
		t.Errorf("expected paper.jar, got %q", resolved.Env["JAR"])
	}
	if resolved.Env["SERVER_MEMORY"] != "2048" {
		t.Errorf("expected 2048, got %q", resolved.Env["SERVER_MEMORY"])
	}
	if len(resolved.ResolvedConfigs) != 1 {
		t.Fatalf("expected 1 resolved config, got %d", len(resolved.ResolvedConfigs))
	}
	if len(resolved.ResolvedConfigs[0].Replace) != 1 {
		t.Fatalf("expected 1 replace in config, got %d", len(resolved.ResolvedConfigs[0].Replace))
	}
}

func TestResolveEgg_ValidationFailure(t *testing.T) {
	egg := &Egg{
		Name:         "test",
		DockerImages: map[string]string{"img": "img"},
		Startup:      "./run.sh",
		Environment: []EggVariable{
			{EnvVariable: "REQUIRED_VAR", Rules: "required"},
		},
		FeatureLimits: FeatureLimits{Memory: 512},
		Scripts: EggScripts{
			Installation: EggInstallationScript{
				Script: "echo", ContainerImage: "debian",
			},
		},
	}

	_, err := ResolveEgg(egg, map[string]string{}, "")
	if err == nil {
		t.Fatal("expected validation error for missing required var")
	}
}
