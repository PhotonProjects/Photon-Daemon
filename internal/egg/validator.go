package egg

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Rule est l'interface que toute règle de validation doit implémenter.
type Rule interface {
	Name() string
	Validate(value string) error
}

// Règles concrètes

type RuleRequired struct{}

func (r RuleRequired) Name() string { return "required" }

func (r RuleRequired) Validate(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("value is required")
	}
	return nil
}

type RuleString struct{}

func (r RuleString) Name() string { return "string" }

func (r RuleString) Validate(value string) error {
	return nil
}

type RuleNumeric struct{}

func (r RuleNumeric) Name() string { return "numeric" }

func (r RuleNumeric) Validate(value string) error {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return fmt.Errorf("must be numeric")
	}
	return nil
}

type RuleInteger struct{}

func (r RuleInteger) Name() string { return "integer" }

func (r RuleInteger) Validate(value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return fmt.Errorf("must be an integer")
	}
	return nil
}

type RuleBoolean struct{}

func (r RuleBoolean) Name() string { return "boolean" }

func (r RuleBoolean) Validate(value string) error {
	switch strings.ToLower(value) {
	case "true", "false", "0", "1":
		return nil
	}
	return fmt.Errorf("must be a boolean")
}

type RuleMax struct {
	Max int
}

func (r RuleMax) Name() string { return "max" }

func (r RuleMax) Validate(value string) error {
	if len(value) > r.Max {
		return fmt.Errorf("must not exceed %d characters", r.Max)
	}
	return nil
}

type RuleMin struct {
	Min int
}

func (r RuleMin) Name() string { return "min" }

func (r RuleMin) Validate(value string) error {
	if len(value) < r.Min {
		return fmt.Errorf("must be at least %d characters", r.Min)
	}
	return nil
}

type RuleRegex struct {
	Pattern *regexp.Regexp
	raw     string
}

func (r RuleRegex) Name() string { return "regex" }

func (r RuleRegex) Validate(value string) error {
	if !r.Pattern.MatchString(value) {
		return fmt.Errorf("must match pattern %s", r.raw)
	}
	return nil
}

// ParseRules convertit une chaîne de règles "required|string|max:50" en []Rule.
func ParseRules(rules string) ([]Rule, error) {
	if strings.TrimSpace(rules) == "" {
		return nil, nil
	}

	parts := strings.Split(rules, "|")
	var parsed []Rule

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, ":", 2)
		ruleName := strings.ToLower(strings.TrimSpace(kv[0]))
		ruleArg := ""
		if len(kv) == 2 {
			ruleArg = strings.TrimSpace(kv[1])
		}

		rule, err := buildRule(ruleName, ruleArg)
		if err != nil {
			return nil, fmt.Errorf("invalid rule %q: %w", part, err)
		}
		parsed = append(parsed, rule)
	}

	return parsed, nil
}

func buildRule(name, arg string) (Rule, error) {
	switch name {
	case "required":
		return RuleRequired{}, nil
	case "string":
		return RuleString{}, nil
	case "numeric":
		return RuleNumeric{}, nil
	case "integer":
		return RuleInteger{}, nil
	case "boolean":
		return RuleBoolean{}, nil
	case "max":
		if arg == "" {
			return nil, fmt.Errorf("max rule requires an argument")
		}
		n, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("max rule argument must be a number")
		}
		return RuleMax{Max: n}, nil
	case "min":
		if arg == "" {
			return nil, fmt.Errorf("min rule requires an argument")
		}
		n, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("min rule argument must be a number")
		}
		return RuleMin{Min: n}, nil
	case "regex":
		if arg == "" {
			return nil, fmt.Errorf("regex rule requires a pattern")
		}
		re, err := regexp.Compile(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return RuleRegex{Pattern: re, raw: arg}, nil
	default:
		return nil, fmt.Errorf("unknown rule %q", name)
	}
}

// ValidateVar applique les règles d'une variable à une valeur.
func ValidateVar(variable EggVariable, value string) error {
	rules, err := ParseRules(variable.Rules)
	if err != nil {
		return fmt.Errorf("failed to parse rules for %s: %w", variable.EnvVariable, err)
	}

	for _, rule := range rules {
		if err := rule.Validate(value); err != nil {
			return fmt.Errorf("%s: %s: %w", variable.EnvVariable, rule.Name(), err)
		}
	}
	return nil
}
