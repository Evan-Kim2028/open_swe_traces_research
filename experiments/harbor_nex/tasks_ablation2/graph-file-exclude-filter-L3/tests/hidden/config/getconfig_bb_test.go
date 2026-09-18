package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	goversion "github.com/hashicorp/go-version"

	"github.com/mgechev/revive/config"
	"github.com/mgechev/revive/lint"
)

func TestGetConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		for name, tc := range map[string]struct {
			confPath   string
			wantConfig lint.Config
		}{
			"default config": {
				wantConfig: lint.Config{
					IgnoreGeneratedHeader: false,
					Confidence:            0.8,
					Severity:              lint.SeverityWarning,
					EnableAllRules:        false,
					EnableDefaultRules:    false,
					Rules: lint.RulesConfig{
						"blank-imports": {
							Severity: lint.SeverityWarning,
						},
						"context-as-argument": {
							Severity: lint.SeverityWarning,
						},
						"context-keys-type": {
							Severity: lint.SeverityWarning,
						},
						"dot-imports": {
							Severity: lint.SeverityWarning,
						},
						"empty-block": {
							Severity: lint.SeverityWarning,
						},
						"error-naming": {
							Severity: lint.SeverityWarning,
						},
						"error-return": {
							Severity: lint.SeverityWarning,
						},
						"error-strings": {
							Severity: lint.SeverityWarning,
						},
						"errorf": {
							Severity: lint.SeverityWarning,
						},
						"exported": {
							Severity: lint.SeverityWarning,
						},
						"increment-decrement": {
							Severity: lint.SeverityWarning,
						},
						"indent-error-flow": {
							Severity: lint.SeverityWarning,
						},
						"package-comments": {
							Severity: lint.SeverityWarning,
						},
						"range": {
							Severity: lint.SeverityWarning,
						},
						"receiver-naming": {
							Severity: lint.SeverityWarning,
						},
						"redefines-builtin-id": {
							Severity: lint.SeverityWarning,
						},
						"superfluous-else": {
							Severity: lint.SeverityWarning,
						},
						"time-naming": {
							Severity: lint.SeverityWarning,
						},
						"unexported-return": {
							Severity: lint.SeverityWarning,
						},
						"unreachable-code": {
							Severity: lint.SeverityWarning,
						},
						"unused-parameter": {
							Severity: lint.SeverityWarning,
						},
						"var-declaration": {
							Severity: lint.SeverityWarning,
						},
						"var-naming": {
							Severity: lint.SeverityWarning,
						},
					},
					ErrorCode:   0,
					WarningCode: 0,
					Directives:  lint.DirectivesConfig{},
					Exclude:     []string{},
					GoVersion:   nil,
				},
			},
			"non-reg issue #470": {
				confPath: "issue-470.toml",
				wantConfig: lint.Config{
					Confidence: 0.8,
					Severity:   lint.SeverityWarning,
					Rules: lint.RulesConfig{
						"add-constant": {
							Severity: lint.SeverityWarning,
							Arguments: lint.Arguments{
								map[string]any{
									"maxLitCount": "3",
									"allowStrs":   `"`,
									"allowFloats": "0.0,1.0,1.,2.0,2.",
									"allowInts":   "0,1,2",
								},
							},
						},
					},
				},
			},
			"config from file issue #585": {
				confPath: "issue-585.toml",
				wantConfig: lint.Config{
					Confidence: 0.0,
					Severity:   lint.SeverityWarning,
				},
			},
			"config from file default confidence issue #585": {
				confPath: "issue-585-default-confidence.toml",
				wantConfig: lint.Config{
					Confidence: 0.8,
					Severity:   lint.SeverityWarning,
				},
			},
			"config from file go-version": {
				confPath: "go-version.toml",
				wantConfig: lint.Config{
					Confidence: 0.8,
					GoVersion:  goversion.Must(goversion.NewSemver("1.20.0")),
				},
			},
			"config from file ignore-generated-header": {
				confPath: "ignore-generated-header.toml",
				wantConfig: lint.Config{
					Confidence:            0.8,
					IgnoreGeneratedHeader: true,
				},
			},
			"config from file enable-default-rules": {
				confPath: "enable-default.toml",
				wantConfig: lint.Config{
					Confidence:            0.8,
					IgnoreGeneratedHeader: false,
					EnableDefaultRules:    true,
					Rules: lint.RulesConfig{
						"blank-imports":        {},
						"context-as-argument":  {},
						"context-keys-type":    {},
						"dot-imports":          {},
						"empty-block":          {},
						"error-naming":         {},
						"error-return":         {},
						"error-strings":        {},
						"errorf":               {},
						"exported":             {},
						"increment-decrement":  {},
						"indent-error-flow":    {},
						"package-comments":     {},
						"range":                {},
						"receiver-naming":      {},
						"redefines-builtin-id": {},
						"superfluous-else":     {},
						"time-naming":          {},
						"unexported-return":    {},
						"unreachable-code":     {},
						"unused-parameter":     {},
						"var-declaration":      {},
						"var-naming":           {},
					},
				},
			},
			"config with non-defaults": {
				confPath: "non-defaults.toml",
				wantConfig: lint.Config{
					Confidence:            0.5,
					Severity:              lint.SeverityError,
					IgnoreGeneratedHeader: true,
					EnableDefaultRules:    true,
					ErrorCode:             2,
					WarningCode:           1,
					Rules: lint.RulesConfig{
						"argument-limit": {
							Severity: lint.SeverityWarning,
							Exclude:  []string{"excluded/file.go"},
							Arguments: lint.Arguments{
								[]any{4},
							},
						},
						"blank-imports": {
							Disabled: true,
							Severity: lint.SeverityError,
						},
						"context-as-argument": {
							Severity: lint.SeverityError,
						},
						"context-keys-type": {
							Severity: lint.SeverityError,
						},
						"dot-imports": {
							Severity: lint.SeverityError,
						},
						"empty-block": {
							Severity: lint.SeverityError,
						},
						"error-naming": {
							Severity: lint.SeverityError,
						},
						"error-return": {
							Severity: lint.SeverityError,
						},
						"error-strings": {
							Severity: lint.SeverityError,
						},
						"errorf": {
							Severity: lint.SeverityError,
						},
						"exported": {
							Severity: lint.SeverityError,
							Arguments: lint.Arguments{
								"check-private-receivers", "disable-stuttering-check",
							},
							Exclude: []string{"excluded/file-exported.go"},
						},
						"increment-decrement": {
							Severity: lint.SeverityError,
						},
						"indent-error-flow": {
							Severity: lint.SeverityError,
						},
						"package-comments": {
							Severity: lint.SeverityError,
						},
						"range": {
							Severity: lint.SeverityError,
						},
						"receiver-naming": {
							Severity: lint.SeverityError,
						},
						"redefines-builtin-id": {
							Severity: lint.SeverityError,
						},
						"superfluous-else": {
							Severity: lint.SeverityError,
						},
						"time-naming": {
							Severity: lint.SeverityError,
						},
						"unexported-return": {
							Severity: lint.SeverityError,
						},
						"unreachable-code": {
							Severity: lint.SeverityError,
						},
						"unused-parameter": {
							Severity: lint.SeverityError,
						},
						"var-declaration": {
							Severity: lint.SeverityError,
						},
						"var-naming": {
							Severity: lint.SeverityError,
						},
					},
				},
			},
		} {
			t.Run(name, func(t *testing.T) {
				var cfgPath string
				if tc.confPath != "" {
					cfgPath = filepath.Join("testdata", tc.confPath)
				}

				cfg, err := config.GetConfig(cfgPath)
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}
				if cfg.IgnoreGeneratedHeader != tc.wantConfig.IgnoreGeneratedHeader {
					t.Errorf("IgnoreGeneratedHeader: expected %v, got %v", tc.wantConfig.IgnoreGeneratedHeader, cfg.IgnoreGeneratedHeader)
				}
				if cfg.Confidence != tc.wantConfig.Confidence {
					t.Errorf("Confidence: expected %v, got %v", tc.wantConfig.Confidence, cfg.Confidence)
				}
				if cfg.Severity != tc.wantConfig.Severity {
					t.Errorf("Severity: expected %v, got %v", tc.wantConfig.Severity, cfg.Severity)
				}
				if cfg.EnableAllRules != tc.wantConfig.EnableAllRules {
					t.Errorf("EnableAllRules: expected %v, got %v", tc.wantConfig.EnableAllRules, cfg.EnableAllRules)
				}
				if cfg.EnableDefaultRules != tc.wantConfig.EnableDefaultRules {
					t.Errorf("EnableDefaultRules: expected %v, got %v", tc.wantConfig.EnableDefaultRules, cfg.EnableDefaultRules)
				}
				if cfg.ErrorCode != tc.wantConfig.ErrorCode {
					t.Errorf("ErrorCode: expected %v, got %v", tc.wantConfig.ErrorCode, cfg.ErrorCode)
				}
				if cfg.WarningCode != tc.wantConfig.WarningCode {
					t.Errorf("WarningCode: expected %v, got %v", tc.wantConfig.WarningCode, cfg.WarningCode)
				}
				if !tc.wantConfig.GoVersion.Equal(cfg.GoVersion) {
					t.Errorf("GoVersion: expected %v, got %v", tc.wantConfig.GoVersion, cfg.GoVersion)
				}

				if len(cfg.Exclude) != len(tc.wantConfig.Exclude) {
					t.Errorf("Exclude length: expected %v, got %v", len(tc.wantConfig.Exclude), len(cfg.Exclude))
				} else {
					for i, exclude := range tc.wantConfig.Exclude {
						if cfg.Exclude[i] != exclude {
							t.Errorf("Exclude[%d]: expected %v, got %v", i, exclude, cfg.Exclude[i])
						}
					}
				}

				if len(cfg.Rules) != len(tc.wantConfig.Rules) {
					t.Errorf("Rules count: expected %v, got %v", len(tc.wantConfig.Rules), len(cfg.Rules))
				}
				for ruleName, wantRule := range tc.wantConfig.Rules {
					gotRule, exists := cfg.Rules[ruleName]
					if !exists {
						t.Errorf("Rule %q: expected to exist, but not found", ruleName)
						continue
					}
					if gotRule.Disabled != wantRule.Disabled {
						t.Errorf("Rule %q Disabled: expected %v, got %v", ruleName, wantRule.Disabled, gotRule.Disabled)
					}
					if gotRule.Severity != wantRule.Severity {
						t.Errorf("Rule %q Severity: expected %v, got %v", ruleName, wantRule.Severity, gotRule.Severity)
					}
					if len(gotRule.Arguments) != len(wantRule.Arguments) {
						t.Errorf("Rule %q Arguments length: expected %v, got %v", ruleName, len(wantRule.Arguments), len(gotRule.Arguments))
					}
					if len(gotRule.Exclude) != len(wantRule.Exclude) {
						t.Errorf("Rule %q Exclude length: expected %v, got %v", ruleName, len(wantRule.Exclude), len(gotRule.Exclude))
					} else {
						for i, wantExclude := range wantRule.Exclude {
							if gotRule.Exclude[i] != wantExclude {
								t.Errorf("Rule %q Exclude[%d]: expected %v, got %v", ruleName, i, wantExclude, gotRule.Exclude[i])
							}
						}
					}
				}
				// Check for unexpected rules in actual config
				for ruleName := range cfg.Rules {
					if _, exists := tc.wantConfig.Rules[ruleName]; !exists {
						t.Errorf("Rule %q: found in actual config but not expected", ruleName)
					}
				}

				if len(cfg.Directives) != len(tc.wantConfig.Directives) {
					t.Errorf("Directives count: expected %v, got %v", len(tc.wantConfig.Directives), len(cfg.Directives))
				}
				for directiveName, wantDirective := range tc.wantConfig.Directives {
					gotDirective, exists := cfg.Directives[directiveName]
					if !exists {
						t.Errorf("Directive %q: expected to exist, but not found", directiveName)
						continue
					}
					if gotDirective.Severity != wantDirective.Severity {
						t.Errorf("Directive %q Severity: expected %v, got %v", directiveName, wantDirective.Severity, gotDirective.Severity)
					}
				}
				// Check for unexpected directives in actual config
				for directiveName := range cfg.Directives {
					if _, exists := tc.wantConfig.Directives[directiveName]; !exists {
						t.Errorf("Directive %q: found in actual config but not expected", directiveName)
					}
				}
			})
		}

		t.Run("rule-level file filter excludes", func(t *testing.T) {
			cfg, err := config.GetConfig("testdata/rule-level-exclude-850.toml")
			if err != nil {
				t.Fatal("should be valid config")
			}
			r1 := cfg.Rules["r1"]
			if len(r1.Exclude) > 0 {
				t.Fatal("r1 should have empty excludes")
			}
			r2 := cfg.Rules["r2"]
			if len(r2.Exclude) != 1 {
				t.Fatal("r2 should have exclude set")
			}
			if !r2.MustExclude("some/file.go") {
				t.Fatal("r2 should be initialized and exclude some/file.go")
			}
			if r2.MustExclude("some/any-other.go") {
				t.Fatal("r2 should not exclude some/any-other.go")
			}
		})
	})

	t.Run("failure", func(t *testing.T) {
		for name, tc := range map[string]struct {
			confPath  string
			wantError string
		}{
			"unknown file": {
				confPath:  "unknown",
				wantError: "cannot read the config file",
			},
			"malformed file": {
				confPath:  "malformed.toml",
				wantError: "cannot parse the config file",
			},
			"invalid exclude pattern": {
				confPath:  "invalid-exclude-pattern.toml",
				wantError: "error in config of rule [var-naming]",
			},
			"enable-all-rules and enable-default-rules both set": {
				confPath:  "enable-all-and-default.toml",
				wantError: "config options enable-all-rules and enable-default-rules cannot be combined",
			},
			"same option with different casing": {
				confPath:  "duplicate-option.toml",
				wantError: "refer to the same option",
			},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := config.GetConfig(filepath.Join("testdata", tc.confPath))

				if err != nil && !strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("Unexpected error: want %q, got: %q", tc.wantError, err)
				}
			})
		}
	})
}
