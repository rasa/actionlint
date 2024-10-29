package actionlint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
	"gopkg.in/yaml.v3"
)

// PathConfig is a configuration for specific file path pattern. This is for values of the "paths" mapping
// in the configuration file.
type PathConfig struct {
	glob glob.Glob
	// Ignore is a list of patterns. They are used for ignoring errors by matching to the error messages.
	// These are similar to the "-ignore" command line option.
	Ignore []*regexp.Regexp
}

// UnmarshalYAML impelements yaml.Unmarshaler. This function partially initializes the PathConfig object
// and is for internal implementation purpose.
func (cfg *PathConfig) UnmarshalYAML(n *yaml.Node) error {
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		switch k.Value {
		case "ignore":
			if v.Kind != yaml.SequenceNode {
				return fmt.Errorf("yaml: \"ignore\" must be a sequence node at line:%d,col:%d", v.Line, v.Column)
			}
			cfg.Ignore = make([]*regexp.Regexp, 0, len(v.Content))
			for _, p := range v.Content {
				r, err := regexp.Compile(p.Value)
				if err != nil {
					return fmt.Errorf("invalid regular expression %q in \"ignore\" at line%d,col:%d: %w", p.Value, v.Line, v.Column, err)
				}
				cfg.Ignore = append(cfg.Ignore, r)
			}
		default:
			return fmt.Errorf("invalid key %q at line:%d,col:%d", k.Value, k.Line, k.Column)
		}
	}
	return nil
}

// Matches returns whether this config is for the given path.
func (cfg *PathConfig) Matches(path string) bool {
	return cfg.glob.Match(path)
}

// Ignores returns whether the given error should be ignored due to the "ignore" configuration.
func (cfg *PathConfig) Ignores(err *Error) bool {
	for _, r := range cfg.Ignore {
		if r.MatchString(err.Message) {
			return true
		}
	}
	return false
}

// PathConfigs is a "paths" mapping in the configuration file. The keys are glob patterns matching to
// file paths relative to the repository root. And the values are the corresponding configurations.
type PathConfigs map[string]*PathConfig

// UnmarshalYAML impelements yaml.Unmarshaler.
//
// https://pkg.go.dev/gopkg.in/yaml.v3?utm_source=godoc#Unmarshaler
func (cfgs *PathConfigs) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return expectedMapping("paths", n)
	}

	ret := make(PathConfigs, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		pat, child := n.Content[i].Value, n.Content[i+1]

		var cfg PathConfig

		g, err := glob.Compile(pat)
		if err != nil {
			return fmt.Errorf("error while processing glob pattern %q in \"paths\" config: %w", pat, err)
		}
		cfg.glob = g

		if err := cfg.UnmarshalYAML(child); err != nil {
			return fmt.Errorf("error while processing \"paths\" config: %w", err)
		}

		if _, ok := ret[pat]; ok {
			return fmt.Errorf("key duplicates within \"paths\" config: %q", pat)
		}
		ret[pat] = &cfg
	}
	*cfgs = ret

	return nil
}

// Config is configuration of actionlint. This struct instance is parsed from "actionlint.yaml"
// file usually put in ".github" directory.
type Config struct {
	// SelfHostedRunner is configuration for self-hosted runner.
	SelfHostedRunner struct {
		// Labels is label names for self-hosted runner.
		Labels []string `yaml:"labels"`
	} `yaml:"self-hosted-runner"`
	// ConfigVariables is names of configuration variables used in the checked workflows. When this value is nil,
	// property names of `vars` context will not be checked. Otherwise actionlint will report a name which is not
	// listed here as undefined config variables.
	// https://docs.github.com/en/actions/learn-github-actions/variables
	ConfigVariables []string `yaml:"config-variables"`
	// Paths is a "paths" mapping in the configuration file. See the document for PathConfigs for more details.
	Paths PathConfigs `yaml:"paths"`
}

// PathConfigsFor returns a list of all PathConfig values matching to the given file path. The path must
// be relative to the root of the project.
func (cfg *Config) PathConfigsFor(path string) []*PathConfig {
	path = filepath.ToSlash(path)

	var ret []*PathConfig
	if cfg != nil {
		for _, c := range cfg.Paths {
			if c.Matches(path) {
				ret = append(ret, c)
			}
		}
	}
	return ret
}

func parseConfig(b []byte, path string) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		msg := strings.ReplaceAll(err.Error(), "\n", " ")
		return nil, fmt.Errorf("could not parse config file %q: %s", path, msg)
	}
	return &c, nil
}

// ReadConfigFile reads actionlint config file (actionlint.yaml) from the given file path.
func ReadConfigFile(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %q: %w", path, err)
	}
	return parseConfig(b, path)
}

// loadRepoConfig reads config file from the repository's .github/actionlint.yml or
// .github/actionlint.yaml.
func loadRepoConfig(root string) (*Config, error) {
	for _, f := range []string{"actionlint.yaml", "actionlint.yml"} {
		path := filepath.Join(root, ".github", f)
		b, err := os.ReadFile(path)
		if err != nil {
			continue // file does not exist
		}
		cfg, err := parseConfig(b, path)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}
	return nil, nil
}

func writeDefaultConfigFile(path string) error {
	b := []byte(`self-hosted-runner:
  # Labels of self-hosted runner in array of strings.
  labels: []

# Configuration variables in array of strings defined in your repository or
# organization. ` + "`null`" + ` means disabling configuration variables check.
# Empty array means no configuration variable is allowed.
config-variables: null

# Configuration for file paths. The keys are glob patterns to match to file
# paths relative to the repository root. The values are the configurations for
# the file paths. The following configurations are available.
#
# "ignore" is an array of regular expression patterns. Matched error messages
# are ignored. This is similar to the "-ignore" command line option.
paths:
#  .github/workflows/**/*.yml:
#    ignore: []
`)
	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("could not write default configuration file at %q: %w", path, err)
	}
	return nil
}
