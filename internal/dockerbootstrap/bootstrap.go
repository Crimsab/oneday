package dockerbootstrap

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	envName                         = ".env"
	envTemplateName                 = ".env.example"
	configName                      = "config.yaml"
	configTemplateName              = "config.example.yaml"
	defaultPort                     = "8788"
	defaultAllowedHosts             = "localhost:8788,127.0.0.1:8788,oneday-gateway:8788"
	bootstrapTokenKey               = "ONEDAY_GATEWAY_BOOTSTRAP_TOKEN"
	allowedHostsKey                 = "ONEDAY_GATEWAY_ALLOWED_HOSTS"
	portKey                         = "ONEDAY_PORT"
	privateFileMode     os.FileMode = 0o600
)

type Result struct {
	CreatedEnv       bool
	CreatedConfig    bool
	GeneratedToken   bool
	UpdatedHostRules bool
}

func Prepare(root string) (Result, error) {
	root, err := validateRoot(root)
	if err != nil {
		return Result{}, err
	}

	var result Result
	result.CreatedConfig, err = ensurePrivateCopy(
		filepath.Join(root, configTemplateName),
		filepath.Join(root, configName),
	)
	if err != nil {
		return Result{}, fmt.Errorf("preparing %s: %w", configName, err)
	}
	result.CreatedEnv, err = ensurePrivateCopy(
		filepath.Join(root, envTemplateName),
		filepath.Join(root, envName),
	)
	if err != nil {
		return Result{}, fmt.Errorf("preparing %s: %w", envName, err)
	}

	envPath := filepath.Join(root, envName)
	content, err := os.ReadFile(envPath)
	if err != nil {
		return Result{}, fmt.Errorf("reading %s: %w", envName, err)
	}
	values := parseEnv(content)
	if strings.TrimSpace(values[bootstrapTokenKey]) == "" {
		token, tokenErr := generateToken()
		if tokenErr != nil {
			return Result{}, tokenErr
		}
		content = replaceEnvValue(content, bootstrapTokenKey, token)
		result.GeneratedToken = true
	}

	port := strings.TrimSpace(values[portKey])
	if port == "" {
		port = defaultPort
	}
	if err := validatePort(port); err != nil {
		return Result{}, err
	}
	allowedHosts := strings.TrimSpace(values[allowedHostsKey])
	if allowedHosts == "" || allowedHosts == defaultAllowedHosts {
		wanted := fmt.Sprintf(
			"localhost:%s,127.0.0.1:%s,oneday-gateway:8788",
			port,
			port,
		)
		if allowedHosts != wanted {
			content = replaceEnvValue(content, allowedHostsKey, wanted)
			result.UpdatedHostRules = true
		}
	}
	if result.GeneratedToken || result.UpdatedHostRules {
		if err := writePrivateFile(envPath, content); err != nil {
			return Result{}, fmt.Errorf("writing %s: %w", envName, err)
		}
	}
	if err := normalizePrivateFiles(root, filepath.Join(root, configName), envPath); err != nil {
		return Result{}, err
	}
	return result, nil
}

func BootstrapToken(root string) (string, error) {
	root, err := validateRoot(root)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.Join(root, envName))
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("no private .env exists; run `oneday docker init` first")
	}
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", envName, err)
	}
	token := strings.TrimSpace(parseEnv(content)[bootstrapTokenKey])
	if token == "" {
		return "", errors.New("no bootstrap token is configured; run `oneday docker init`")
	}
	return token, nil
}

func validateRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolving installation directory: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("opening installation directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("installation path is not a directory: %s", absolute)
	}
	return absolute, nil
}

func ensurePrivateCopy(source, destination string) (bool, error) {
	if _, err := os.Stat(destination); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return false, fmt.Errorf("reading template %s: %w", filepath.Base(source), err)
	}
	if err := os.WriteFile(destination, content, privateFileMode); err != nil {
		return false, err
	}
	if err := os.Chmod(destination, privateFileMode); err != nil {
		return false, err
	}
	return true, nil
}

func generateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating bootstrap token: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func validatePort(raw string) error {
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%s must be an integer from 1 to 65535", portKey)
	}
	return nil
}

func parseEnv(content []byte) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values
}

func replaceEnvValue(content []byte, key, value string) []byte {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	output := make([]string, 0, len(lines)+1)
	replaced := false
	for _, line := range lines {
		candidate := strings.TrimSpace(line)
		candidateKey, _, found := strings.Cut(candidate, "=")
		if found && strings.TrimSpace(candidateKey) == key {
			if !replaced {
				output = append(output, key+"="+value)
				replaced = true
			}
			continue
		}
		output = append(output, line)
	}
	if !replaced {
		if len(output) > 0 && output[len(output)-1] != "" {
			output = append(output, "")
		}
		output = append(output, key+"="+value)
	}
	return []byte(strings.TrimRight(strings.Join(output, "\n"), "\n") + "\n")
}

func writePrivateFile(path string, content []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".env.oneday-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(privateFileMode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return os.Chmod(path, privateFileMode)
}
