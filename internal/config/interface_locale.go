package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// UpdateInterfaceLocale changes only the interface.locale YAML field while
// preserving unrelated configuration, comments, permissions, and ownership.
func UpdateInterfaceLocale(path, locale string) error {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "auto" {
		locale = ""
	}
	if locale != "" && locale != "en" && locale != "it" {
		return fmt.Errorf("unsupported interface locale %q", locale)
	}
	return withConfigLock(path, func() error {
		raw, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		var doc yaml.Node
		if len(raw) == 0 {
			raw = []byte("config_version: 3\n")
		}
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return err
		}
		if err := rejectDuplicateMappingKeys(&doc, ""); err != nil {
			return err
		}
		root, err := documentRoot(&doc)
		if err != nil {
			return err
		}
		if err := setString(root, locale, "interface", "locale"); err != nil {
			return err
		}
		var out strings.Builder
		encoder := yaml.NewEncoder(&out)
		encoder.SetIndent(2)
		if err := encoder.Encode(&doc); err != nil {
			return err
		}
		if err := encoder.Close(); err != nil {
			return err
		}
		next := []byte(out.String())
		if _, err := configFromEditBytes(path, next); err != nil {
			return err
		}
		return writeConfigAtomic(path, raw, next)
	})
}
