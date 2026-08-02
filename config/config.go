// Package config loads, discovers, validates, and resolves TagLock policy.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/theworker02/taglock/rule"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

const CurrentVersion = 1

var Filenames = []string{".taglock.yml", ".taglock.yaml", "taglock.yml", "taglock.yaml"}

// Config is a normalized policy. The legacy exported fields remain supported
// for Phase 1 embedders; new code should use NamespacePolicies and Security.
type Config struct {
	Version           int
	Severity          map[string]string
	Rules             RuleSelection
	NamespacePolicies map[string]Namespace
	CrossNamespace    CrossNamespace
	Security          Security
	Overrides         []Override
	Go                GoPolicy
	JSON              JSONPolicy
	Contracts         ContractsPolicy
	Compatibility     CompatibilityPolicy
	Evolution         EvolutionPolicy
	Snapshot          SnapshotPolicy
	Schema            SchemaPolicy
	Verification      VerificationPolicy

	// Phase 1 compatibility fields.
	Namespaces             []string
	Naming                 map[string]string
	SensitiveFields        []string
	RequireExplicitJSON    bool
	DisallowUnknownOptions bool
}

type RuleSelection struct {
	Enable  []string `yaml:"enable"`
	Disable []string `yaml:"disable"`
}

type Namespace struct {
	Enabled             bool     `yaml:"enabled"`
	Naming              string   `yaml:"naming,omitempty"`
	RequireExplicitTags bool     `yaml:"require_explicit_tags,omitempty"`
	KnownOptions        []string `yaml:"known_options,omitempty"`
	CaseInsensitive     bool     `yaml:"case_insensitive,omitempty"`
}

type CrossNamespace struct {
	Compare     []string `yaml:"compare"`
	NamingDrift string   `yaml:"naming_drift,omitempty"`
}

type Security struct {
	SensitiveFields  []string `yaml:"sensitive_fields"`
	OutputNamespaces []string `yaml:"output_namespaces"`
	InputNamespaces  []string `yaml:"input_namespaces,omitempty"`
	RequireWriteOnly bool     `yaml:"require_write_only,omitempty"`
}

type Override struct {
	Files    []string          `yaml:"files,omitempty"`
	Packages []string          `yaml:"packages,omitempty"`
	Rules    RuleSelection     `yaml:"rules,omitempty"`
	Severity map[string]string `yaml:"severity,omitempty"`
}

type GoPolicy struct {
	MinimumVersion string   `yaml:"minimum_version,omitempty"`
	BuildTags      []string `yaml:"build_tags,omitempty"`
}
type JSONPolicy struct {
	Semantics         string   `yaml:"semantics,omitempty"`
	SupportedProfiles []string `yaml:"supported_profiles,omitempty"`
	CompareProfiles   bool     `yaml:"compare_profiles,omitempty"`
	MigrationCheck    bool     `yaml:"migration_check,omitempty"`
}
type ContractsPolicy struct {
	Include         []string          `yaml:"include,omitempty"`
	Exclude         []string          `yaml:"exclude,omitempty"`
	Visibility      []string          `yaml:"visibility,omitempty"`
	DefaultLevel    string            `yaml:"default_level,omitempty"`
	Annotations     AnnotationsPolicy `yaml:"annotations,omitempty"`
	PackageDefaults map[string]string `yaml:"package_defaults,omitempty"`
}
type AnnotationsPolicy struct {
	Enabled bool `yaml:"enabled"`
}
type CompatibilityPolicy struct {
	DefaultMode    string                        `yaml:"default_mode,omitempty"`
	FailOn         []string                      `yaml:"fail_on,omitempty"`
	ContractLevels map[string]LevelCompatibility `yaml:"contract_levels,omitempty"`
}
type LevelCompatibility struct {
	Policy string `yaml:"policy"`
}
type EvolutionPolicy struct {
	RequireDeprecationBeforeRemoval   bool     `yaml:"require_deprecation_before_removal,omitempty"`
	MinimumDeprecationReleases        int      `yaml:"minimum_deprecation_releases,omitempty"`
	RequireReplacementForPublicFields bool     `yaml:"require_replacement_for_public_fields,omitempty"`
	ApprovalsFile                     string   `yaml:"approvals_file,omitempty"`
	ReleaseHistory                    []string `yaml:"release_history,omitempty"`
}
type SnapshotPolicy struct {
	Output       string `yaml:"output,omitempty"`
	Reproducible bool   `yaml:"reproducible,omitempty"`
}
type SchemaPolicy struct {
	Enabled    bool   `yaml:"enabled"`
	Format     string `yaml:"format,omitempty"`
	Output     string `yaml:"output,omitempty"`
	CheckDrift bool   `yaml:"check_drift,omitempty"`
}
type VerificationPolicy struct {
	Enabled           bool                           `yaml:"enabled"`
	FixturesDirectory string                         `yaml:"fixtures_directory,omitempty"`
	Fixtures          map[string]VerificationFixture `yaml:"fixtures,omitempty"`
}
type VerificationFixture struct {
	File          string   `yaml:"file"`
	TypeArguments []string `yaml:"type_arguments,omitempty"`
}

type rawConfig struct {
	Version        *int                    `yaml:"version"`
	Severity       map[string]string       `yaml:"severity"`
	Rules          *RuleSelection          `yaml:"rules"`
	Namespaces     map[string]rawNamespace `yaml:"namespaces"`
	CrossNamespace *rawCrossNamespace      `yaml:"cross_namespace"`
	Security       *rawSecurity            `yaml:"security"`
	Overrides      []rawOverride           `yaml:"overrides"`
	Go             *GoPolicy               `yaml:"go"`
	JSON           *JSONPolicy             `yaml:"json"`
	Contracts      *ContractsPolicy        `yaml:"contracts"`
	Compatibility  *CompatibilityPolicy    `yaml:"compatibility"`
	Evolution      *EvolutionPolicy        `yaml:"evolution"`
	Snapshot       *SnapshotPolicy         `yaml:"snapshot"`
	Schema         *SchemaPolicy           `yaml:"schema"`
	Verification   *VerificationPolicy     `yaml:"verification"`
}

type rawNamespace struct {
	Enabled             *bool    `yaml:"enabled"`
	Naming              string   `yaml:"naming"`
	RequireExplicitTags bool     `yaml:"require_explicit_tags"`
	KnownOptions        []string `yaml:"known_options"`
	CaseInsensitive     *bool    `yaml:"case_insensitive"`
}

type rawCrossNamespace struct {
	Compare     []string `yaml:"compare"`
	NamingDrift string   `yaml:"naming_drift"`
}

type rawSecurity struct {
	SensitiveFields  []string `yaml:"sensitive_fields"`
	OutputNamespaces []string `yaml:"output_namespaces"`
	InputNamespaces  []string `yaml:"input_namespaces"`
	RequireWriteOnly bool     `yaml:"require_write_only"`
}

type rawOverride struct {
	Files    []string          `yaml:"files"`
	Packages []string          `yaml:"packages"`
	Rules    RuleSelection     `yaml:"rules"`
	Severity map[string]string `yaml:"severity"`
}

type legacyConfig struct {
	Namespaces             *[]string          `yaml:"namespaces"`
	Naming                 *map[string]string `yaml:"naming"`
	SensitiveFields        *[]string          `yaml:"sensitive_fields"`
	RequireExplicitJSON    *bool              `yaml:"require_explicit_json_tags"`
	DisallowUnknownOptions *bool              `yaml:"disallow_unknown_options"`
}

// Default returns the versioned built-in policy.
func Default() Config {
	sensitive := []string{"password", "password_hash", "passwd", "secret", "token", "access_token", "refresh_token", "api_key", "private_key", "client_secret", "credential", "authorization", "session_id"}
	policies := map[string]Namespace{
		"json":         {Enabled: true, CaseInsensitive: true},
		"yaml":         {Enabled: true},
		"xml":          {Enabled: true},
		"db":           {Enabled: true},
		"validate":     {Enabled: true},
		"form":         {Enabled: true},
		"query":        {Enabled: true},
		"sql":          {Enabled: true},
		"mapstructure": {Enabled: true},
	}
	return Config{
		Version: CurrentVersion, Severity: map[string]string{}, NamespacePolicies: policies,
		CrossNamespace: CrossNamespace{Compare: []string{"json", "yaml", "xml", "db", "form"}, NamingDrift: "warning"},
		Security:       Security{SensitiveFields: append([]string(nil), sensitive...), OutputNamespaces: []string{"json", "yaml", "xml"}, InputNamespaces: []string{"form", "query"}},
		Namespaces:     []string{"json", "yaml", "xml", "db", "validate", "form", "query", "sql", "mapstructure"}, Naming: map[string]string{},
		SensitiveFields: append([]string(nil), sensitive...), DisallowUnknownOptions: true,
		Go:            GoPolicy{MinimumVersion: "1.24"},
		JSON:          JSONPolicy{Semantics: "auto", SupportedProfiles: []string{"json/v1", "json/v2"}, CompareProfiles: true},
		Contracts:     ContractsPolicy{Visibility: []string{"exported"}, DefaultLevel: "public", Annotations: AnnotationsPolicy{Enabled: true}, PackageDefaults: map[string]string{}},
		Compatibility: CompatibilityPolicy{DefaultMode: "bidirectional", FailOn: []string{"breaking", "unknown"}, ContractLevels: map[string]LevelCompatibility{"public": {Policy: "strict"}, "internal": {Policy: "relaxed"}, "storage": {Policy: "storage"}, "experimental": {Policy: "advisory"}}},
		Evolution:     EvolutionPolicy{ApprovalsFile: ".taglock/changes.yml"},
		Snapshot:      SnapshotPolicy{Output: ".taglock/contracts.json", Reproducible: true},
		Schema:        SchemaPolicy{Format: "json-schema", Output: "schemas/contracts.json"},
		Verification:  VerificationPolicy{FixturesDirectory: "testdata/contracts", Fixtures: map[string]VerificationFixture{}},
	}
}

// Load reads a configuration file.
func Load(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	cfg, err := Decode(file)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", filename, err)
	}
	return cfg, nil
}

// Decode accepts both the Phase 2 schema and the legacy Phase 1 schema.
func Decode(reader io.Reader) (Config, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return Config{}, fmt.Errorf("read TagLock config: %w", err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return Config{}, fmt.Errorf("decode TagLock config: %w", err)
	}
	legacy := namespaceNodeIsSequence(&document)
	if legacy {
		return decodeLegacy(data)
	}
	var raw rawConfig
	if err := strictDecode(data, &raw); err != nil {
		return Config{}, fmt.Errorf("decode TagLock config: %w", err)
	}
	cfg := Default()
	if raw.Version != nil {
		cfg.Version = *raw.Version
	}
	if raw.Severity != nil {
		cfg.Severity = cloneMap(raw.Severity)
	}
	if raw.Rules != nil {
		cfg.Rules = *raw.Rules
	}
	if raw.Namespaces != nil {
		cfg.NamespacePolicies = make(map[string]Namespace, len(raw.Namespaces))
		for name, item := range raw.Namespaces {
			enabled := true
			if item.Enabled != nil {
				enabled = *item.Enabled
			}
			caseInsensitive := false
			if item.CaseInsensitive != nil {
				caseInsensitive = *item.CaseInsensitive
			}
			cfg.NamespacePolicies[name] = Namespace{Enabled: enabled, Naming: item.Naming, RequireExplicitTags: item.RequireExplicitTags, KnownOptions: append([]string(nil), item.KnownOptions...), CaseInsensitive: caseInsensitive}
		}
	}
	if raw.Namespaces != nil && raw.CrossNamespace == nil {
		cfg.CrossNamespace.Compare = filterEnabled(cfg, cfg.CrossNamespace.Compare)
	}
	if raw.Namespaces != nil && raw.Security == nil {
		cfg.Security.OutputNamespaces = filterEnabled(cfg, cfg.Security.OutputNamespaces)
		cfg.Security.InputNamespaces = filterEnabled(cfg, cfg.Security.InputNamespaces)
	}
	if raw.CrossNamespace != nil {
		cfg.CrossNamespace = CrossNamespace{Compare: append([]string(nil), raw.CrossNamespace.Compare...), NamingDrift: raw.CrossNamespace.NamingDrift}
	}
	if raw.Security != nil {
		cfg.Security = Security{SensitiveFields: append([]string(nil), raw.Security.SensitiveFields...), OutputNamespaces: append([]string(nil), raw.Security.OutputNamespaces...), InputNamespaces: append([]string(nil), raw.Security.InputNamespaces...), RequireWriteOnly: raw.Security.RequireWriteOnly}
	}
	if raw.Overrides != nil {
		cfg.Overrides = make([]Override, 0, len(raw.Overrides))
		for _, item := range raw.Overrides {
			cfg.Overrides = append(cfg.Overrides, Override{Files: append([]string(nil), item.Files...), Packages: append([]string(nil), item.Packages...), Rules: item.Rules, Severity: cloneMap(item.Severity)})
		}
	}
	if raw.Go != nil {
		cfg.Go = *raw.Go
	}
	if raw.JSON != nil {
		cfg.JSON = *raw.JSON
	}
	if raw.Contracts != nil {
		cfg.Contracts = *raw.Contracts
	}
	if raw.Compatibility != nil {
		cfg.Compatibility = *raw.Compatibility
	}
	if raw.Evolution != nil {
		cfg.Evolution = *raw.Evolution
	}
	if raw.Snapshot != nil {
		cfg.Snapshot = *raw.Snapshot
	}
	if raw.Schema != nil {
		cfg.Schema = *raw.Schema
	}
	if raw.Verification != nil {
		cfg.Verification = *raw.Verification
	}
	cfg.syncLegacy()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func decodeLegacy(data []byte) (Config, error) {
	var raw legacyConfig
	if err := strictDecode(data, &raw); err != nil {
		return Config{}, fmt.Errorf("decode legacy TagLock config: %w", err)
	}
	cfg := Default()
	if raw.Namespaces != nil {
		cfg.NamespacePolicies = map[string]Namespace{}
		for _, name := range *raw.Namespaces {
			cfg.NamespacePolicies[name] = Namespace{Enabled: true}
		}
	}
	cfg.CrossNamespace.Compare = filterEnabled(cfg, cfg.CrossNamespace.Compare)
	cfg.Security.OutputNamespaces = filterEnabled(cfg, cfg.Security.OutputNamespaces)
	cfg.Security.InputNamespaces = filterEnabled(cfg, cfg.Security.InputNamespaces)
	if raw.Naming != nil {
		for name, naming := range *raw.Naming {
			item := cfg.NamespacePolicies[name]
			item.Naming = naming
			cfg.NamespacePolicies[name] = item
		}
	}
	if raw.SensitiveFields != nil {
		cfg.Security.SensitiveFields = append([]string(nil), (*raw.SensitiveFields)...)
	}
	if raw.RequireExplicitJSON != nil {
		item := cfg.NamespacePolicies["json"]
		item.RequireExplicitTags = *raw.RequireExplicitJSON
		cfg.NamespacePolicies["json"] = item
	}
	if raw.DisallowUnknownOptions != nil {
		cfg.DisallowUnknownOptions = *raw.DisallowUnknownOptions
	}
	cfg.syncLegacy()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func strictDecode(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	return decoder.Decode(target)
}

func namespaceNodeIsSequence(document *yaml.Node) bool {
	if len(document.Content) == 0 {
		return false
	}
	root := document.Content[0]
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == "namespaces" {
			return root.Content[index+1].Kind == yaml.SequenceNode
		}
	}
	return false
}

// Validate rejects ambiguous or ineffective policy.
func (c Config) Validate() error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version %d (supported: %d)", c.Version, CurrentVersion)
	}
	if len(c.NamespacePolicies) == 0 {
		return errors.New("configuration must define at least one namespace")
	}
	for name, namespace := range c.NamespacePolicies {
		if strings.TrimSpace(name) != name || name == "" || strings.ContainsAny(name, " ,:\t\r\n") {
			return fmt.Errorf("invalid namespace name %q", name)
		}
		if namespace.Naming != "" && !validNaming(namespace.Naming) {
			return fmt.Errorf("namespace %s has unsupported naming style %q", name, namespace.Naming)
		}
		seen := map[string]bool{}
		for _, option := range namespace.KnownOptions {
			if option == "" || strings.Contains(option, ",") {
				return fmt.Errorf("namespace %s has invalid known option %q", name, option)
			}
			if seen[option] {
				return fmt.Errorf("namespace %s repeats known option %q", name, option)
			}
			seen[option] = true
		}
	}
	for id, value := range c.Severity {
		if !rule.Known(id) {
			return fmt.Errorf("severity references unknown rule %q", id)
		}
		if _, err := rule.ParseSeverity(value); err != nil {
			return fmt.Errorf("severity for %s: %w", id, err)
		}
	}
	if err := validateRuleSelection(c.Rules); err != nil {
		return err
	}
	if c.CrossNamespace.NamingDrift != "" {
		if _, err := rule.ParseSeverity(c.CrossNamespace.NamingDrift); err != nil {
			return fmt.Errorf("cross_namespace.naming_drift: %w", err)
		}
	}
	for _, name := range append(append([]string(nil), c.CrossNamespace.Compare...), append(c.Security.OutputNamespaces, c.Security.InputNamespaces...)...) {
		if !c.Enabled(name) {
			return fmt.Errorf("policy references disabled or unknown namespace %q", name)
		}
	}
	for index, override := range c.Overrides {
		if len(override.Files) == 0 && len(override.Packages) == 0 {
			return fmt.Errorf("override %d must specify files or packages", index)
		}
		for _, pattern := range append(append([]string(nil), override.Files...), override.Packages...) {
			if err := validateGlob(pattern); err != nil {
				return fmt.Errorf("override %d has malformed glob %q: %w", index, pattern, err)
			}
		}
		if err := validateRuleSelection(override.Rules); err != nil {
			return fmt.Errorf("override %d: %w", index, err)
		}
		for id, severity := range override.Severity {
			if !rule.Known(id) {
				return fmt.Errorf("override %d references unknown rule %q", index, id)
			}
			if _, err := rule.ParseSeverity(severity); err != nil {
				return fmt.Errorf("override %d severity for %s: %w", index, id, err)
			}
		}
	}
	if c.Go.MinimumVersion != "" && !validGoVersion(c.Go.MinimumVersion) {
		return fmt.Errorf("go.minimum_version %q must use major.minor form", c.Go.MinimumVersion)
	}
	switch c.JSON.Semantics {
	case "", "auto", "v1", "v2", "both":
	default:
		return fmt.Errorf("json.semantics has unknown value %q", c.JSON.Semantics)
	}
	for _, profile := range c.JSON.SupportedProfiles {
		if profile != "json/v1" && profile != "json/v2" {
			return fmt.Errorf("json.supported_profiles contains unknown profile %q", profile)
		}
	}
	levels := map[string]bool{"public": true, "internal": true, "storage": true, "message": true, "configuration": true, "experimental": true}
	if !levels[c.Contracts.DefaultLevel] {
		return fmt.Errorf("contracts.default_level has invalid level %q", c.Contracts.DefaultLevel)
	}
	for pattern, level := range c.Contracts.PackageDefaults {
		if err := validateGlob(pattern); err != nil {
			return fmt.Errorf("contracts.package_defaults has malformed glob %q: %w", pattern, err)
		}
		if !levels[level] {
			return fmt.Errorf("contracts.package_defaults uses invalid level %q", level)
		}
	}
	for _, pattern := range append(append([]string(nil), c.Contracts.Include...), c.Contracts.Exclude...) {
		if err := validateGlob(pattern); err != nil {
			return fmt.Errorf("contracts path pattern %q is invalid: %w", pattern, err)
		}
	}
	modes := map[string]bool{"producer": true, "consumer": true, "bidirectional": true, "storage": true, "configuration": true, "message": true, "advisory": true}
	if !modes[c.Compatibility.DefaultMode] {
		return fmt.Errorf("compatibility.default_mode has invalid mode %q", c.Compatibility.DefaultMode)
	}
	for level, item := range c.Compatibility.ContractLevels {
		if !levels[level] {
			return fmt.Errorf("compatibility references invalid contract level %q", level)
		}
		if item.Policy != "strict" && item.Policy != "relaxed" && item.Policy != "storage" && item.Policy != "advisory" {
			return fmt.Errorf("compatibility level %s has invalid policy %q", level, item.Policy)
		}
	}
	for _, value := range c.Compatibility.FailOn {
		if value != "breaking" && value != "potentially-breaking" && value != "unknown" {
			return fmt.Errorf("compatibility.fail_on has invalid severity %q", value)
		}
	}
	if c.Evolution.MinimumDeprecationReleases < 0 {
		return errors.New("evolution.minimum_deprecation_releases cannot be negative")
	}
	previousRelease := ""
	seenReleases := map[string]bool{}
	for _, release := range c.Evolution.ReleaseHistory {
		normalized := normalizeSemver(release)
		if !semver.IsValid(normalized) {
			return fmt.Errorf("evolution.release_history contains invalid semantic version %q", release)
		}
		if seenReleases[normalized] {
			return fmt.Errorf("evolution.release_history repeats version %q", release)
		}
		if previousRelease != "" && semver.Compare(previousRelease, normalized) >= 0 {
			return errors.New("evolution.release_history must be strictly increasing")
		}
		seenReleases[normalized] = true
		previousRelease = normalized
	}
	for identity, fixture := range c.Verification.Fixtures {
		if strings.TrimSpace(identity) == "" {
			return errors.New("verification.fixtures contains an empty contract identity")
		}
		if fixture.File != "" && filepath.IsAbs(fixture.File) {
			return fmt.Errorf("verification fixture %q must use a project-relative file", identity)
		}
		for _, argument := range fixture.TypeArguments {
			if strings.TrimSpace(argument) == "" {
				return fmt.Errorf("verification fixture %q contains an empty type argument", identity)
			}
			if _, err := parser.ParseExpr(argument); err != nil {
				return fmt.Errorf("verification fixture %q has invalid type argument %q: %w", identity, argument, err)
			}
		}
	}
	if c.Schema.Format != "" && c.Schema.Format != "json-schema" && c.Schema.Format != "openapi" {
		return fmt.Errorf("schema.format has invalid value %q", c.Schema.Format)
	}
	return nil
}

func normalizeSemver(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	return value
}

func validGoVersion(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	return true
}

func validateRuleSelection(selection RuleSelection) error {
	seen := map[string]string{}
	for _, pair := range []struct {
		name string
		ids  []string
	}{{"enable", selection.Enable}, {"disable", selection.Disable}} {
		for _, id := range pair.ids {
			if !rule.Known(id) {
				return fmt.Errorf("rules.%s references unknown rule %q", pair.name, id)
			}
			if prior := seen[id]; prior != "" {
				return fmt.Errorf("rule %s appears in both or repeatedly in %s and %s", id, prior, pair.name)
			}
			seen[id] = pair.name
		}
	}
	return nil
}

func validateGlob(pattern string) error {
	_, err := path.Match(strings.ReplaceAll(filepath.ToSlash(pattern), "**", "*"), "probe")
	return err
}
func validNaming(value string) bool {
	switch value {
	case "snake_case", "camelCase", "PascalCase", "kebab-case":
		return true
	}
	return false
}

// Discover walks upward and returns the first supported configuration path.
func Discover(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Stat(directory); statErr == nil && !info.IsDir() {
		directory = filepath.Dir(directory)
	}
	for {
		for _, name := range Filenames {
			candidate := filepath.Join(directory, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", nil
		}
		directory = parent
	}
}

// Enabled reports whether namespace participates in analysis.
func (c Config) Enabled(name string) bool {
	if c.NamespacePolicies != nil {
		item, ok := c.NamespacePolicies[name]
		return ok && item.Enabled
	}
	for _, legacy := range c.Namespaces {
		if legacy == name {
			return true
		}
	}
	return false
}

func (c Config) Namespace(name string) Namespace {
	item := c.NamespacePolicies[name]
	if naming := c.Naming[name]; item.Naming == "" && naming != "" {
		item.Naming = naming
	}
	if name == "json" && c.RequireExplicitJSON {
		item.RequireExplicitTags = true
	}
	return item
}

func (c Config) SortedNamespaces() []string {
	result := []string{}
	for name, item := range c.NamespacePolicies {
		if item.Enabled {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func (c Config) RuleEnabled(id string) bool {
	definition, ok := rule.Lookup(id)
	if !ok {
		return false
	}
	enabled := definition.DefaultEnabled
	for _, candidate := range c.Rules.Enable {
		if candidate == id {
			enabled = true
		}
	}
	for _, candidate := range c.Rules.Disable {
		if candidate == id {
			enabled = false
		}
	}
	if severity, ok := c.Severity[id]; ok && severity == "off" {
		enabled = false
	}
	return enabled
}

func (c Config) RuleSeverity(id string) rule.Severity {
	definition, ok := rule.Lookup(id)
	if !ok {
		return rule.SeverityOff
	}
	if configured, ok := c.Severity[id]; ok {
		severity, _ := rule.ParseSeverity(configured)
		return severity
	}
	if id == "TAG301" && c.CrossNamespace.NamingDrift != "" {
		severity, _ := rule.ParseSeverity(c.CrossNamespace.NamingDrift)
		return severity
	}
	return definition.DefaultSeverity
}

// Effective applies matching file/package overrides without mutating c.
func (c Config) Effective(filename, packagePath string) Config {
	result := c.Clone()
	slashFile := filepath.ToSlash(filename)
	for _, override := range c.Overrides {
		matched := matchesAny(override.Files, slashFile) || matchesAny(override.Packages, packagePath)
		if !matched {
			continue
		}
		result.Rules.Enable = append(result.Rules.Enable, override.Rules.Enable...)
		result.Rules.Disable = append(result.Rules.Disable, override.Rules.Disable...)
		for id, severity := range override.Severity {
			result.Severity[id] = severity
		}
	}
	return result
}

// SelectContract applies snapshot include/exclude and visibility policy.
func (c Config) SelectContract(packagePath, filename string, exported bool) bool {
	value := filepath.ToSlash(filename)
	if len(c.Contracts.Include) > 0 && !matchesAny(c.Contracts.Include, packagePath) && !matchesAny(c.Contracts.Include, value) {
		return false
	}
	if matchesAny(c.Contracts.Exclude, packagePath) || matchesAny(c.Contracts.Exclude, value) {
		return false
	}
	if len(c.Contracts.Visibility) > 0 && !exported {
		allowed := false
		for _, visibility := range c.Contracts.Visibility {
			if visibility == "all" {
				allowed = true
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

// ContractLevel resolves an explicit annotation or package default.
func (c Config) ContractLevel(packagePath, explicit string) string {
	if explicit != "" && c.Contracts.Annotations.Enabled {
		return explicit
	}
	patterns := make([]string, 0, len(c.Contracts.PackageDefaults))
	for pattern := range c.Contracts.PackageDefaults {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)
	for _, pattern := range patterns {
		if matchesAny([]string{pattern}, packagePath) {
			return c.Contracts.PackageDefaults[pattern]
		}
	}
	return c.Contracts.DefaultLevel
}

func ValidContractLevel(value string) bool {
	switch value {
	case "public", "internal", "storage", "message", "configuration", "experimental":
		return true
	}
	return false
}

func matchesAny(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if matchGlob(filepath.ToSlash(pattern), value) {
			return true
		}
	}
	return false
}
func matchGlob(pattern, value string) bool {
	patternParts, valueParts := strings.Split(strings.Trim(pattern, "/"), "/"), strings.Split(strings.Trim(value, "/"), "/")
	var match func(int, int) bool
	match = func(pi, vi int) bool {
		if pi == len(patternParts) {
			return vi == len(valueParts)
		}
		if patternParts[pi] == "**" {
			for next := vi; next <= len(valueParts); next++ {
				if match(pi+1, next) {
					return true
				}
			}
			return false
		}
		if vi == len(valueParts) {
			return false
		}
		ok, _ := path.Match(patternParts[pi], valueParts[vi])
		return ok && match(pi+1, vi+1)
	}
	return match(0, 0)
}

func (c Config) Clone() Config {
	result := c
	result.Severity = cloneMap(c.Severity)
	result.NamespacePolicies = make(map[string]Namespace, len(c.NamespacePolicies))
	for name, item := range c.NamespacePolicies {
		item.KnownOptions = append([]string(nil), item.KnownOptions...)
		result.NamespacePolicies[name] = item
	}
	result.Rules.Enable = append([]string(nil), c.Rules.Enable...)
	result.Rules.Disable = append([]string(nil), c.Rules.Disable...)
	result.Security.SensitiveFields = append([]string(nil), c.Security.SensitiveFields...)
	result.JSON.SupportedProfiles = append([]string(nil), c.JSON.SupportedProfiles...)
	result.Contracts.Include = append([]string(nil), c.Contracts.Include...)
	result.Contracts.Exclude = append([]string(nil), c.Contracts.Exclude...)
	result.Contracts.PackageDefaults = cloneMap(c.Contracts.PackageDefaults)
	result.Evolution.ReleaseHistory = append([]string(nil), c.Evolution.ReleaseHistory...)
	result.Compatibility.ContractLevels = make(map[string]LevelCompatibility, len(c.Compatibility.ContractLevels))
	for key, value := range c.Compatibility.ContractLevels {
		result.Compatibility.ContractLevels[key] = value
	}
	result.Verification.Fixtures = make(map[string]VerificationFixture, len(c.Verification.Fixtures))
	for key, value := range c.Verification.Fixtures {
		value.TypeArguments = append([]string(nil), value.TypeArguments...)
		result.Verification.Fixtures[key] = value
	}
	return result
}

func (c *Config) syncLegacy() {
	c.Namespaces = c.SortedNamespaces()
	c.Naming = map[string]string{}
	for name, item := range c.NamespacePolicies {
		if item.Naming != "" {
			c.Naming[name] = item.Naming
		}
	}
	c.SensitiveFields = append([]string(nil), c.Security.SensitiveFields...)
	c.RequireExplicitJSON = c.NamespacePolicies["json"].RequireExplicitTags
}
func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func filterEnabled(cfg Config, values []string) []string {
	result := values[:0]
	for _, value := range values {
		if cfg.Enabled(value) {
			result = append(result, value)
		}
	}
	return result
}

// Marshal returns deterministic human-readable YAML for config print/init.
func Marshal(c Config) ([]byte, error) {
	type printable struct {
		Version        int                  `yaml:"version"`
		Severity       map[string]string    `yaml:"severity,omitempty"`
		Rules          RuleSelection        `yaml:"rules,omitempty"`
		Namespaces     map[string]Namespace `yaml:"namespaces"`
		CrossNamespace CrossNamespace       `yaml:"cross_namespace"`
		Security       Security             `yaml:"security"`
		Overrides      []Override           `yaml:"overrides,omitempty"`
		Go             GoPolicy             `yaml:"go,omitempty"`
		JSON           JSONPolicy           `yaml:"json,omitempty"`
		Contracts      ContractsPolicy      `yaml:"contracts,omitempty"`
		Compatibility  CompatibilityPolicy  `yaml:"compatibility,omitempty"`
		Evolution      EvolutionPolicy      `yaml:"evolution,omitempty"`
		Snapshot       SnapshotPolicy       `yaml:"snapshot,omitempty"`
		Schema         SchemaPolicy         `yaml:"schema,omitempty"`
		Verification   VerificationPolicy   `yaml:"verification,omitempty"`
	}
	return yaml.Marshal(printable{c.Version, c.Severity, c.Rules, c.NamespacePolicies, c.CrossNamespace, c.Security, c.Overrides, c.Go, c.JSON, c.Contracts, c.Compatibility, c.Evolution, c.Snapshot, c.Schema, c.Verification})
}
