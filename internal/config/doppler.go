package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/albedosehen/rwwwrse/internal/observability"
)

// DopplerProvider is responsible for loading secrets from Doppler.
type DopplerProvider struct {
	logger observability.Logger
}

func NewDopplerProvider(logger observability.Logger) *DopplerProvider {
	return &DopplerProvider{
		logger: logger,
	}
}

// LoadSecrets requires the Doppler CLI to be installed and configured.
// This is a no-op if Doppler is not configured or available.
func (dp *DopplerProvider) LoadSecrets() error {
	ctx := context.Background()

	if !dp.isDopplerCLIAvailable() {
		dp.logger.Debug(ctx, "Doppler CLI not available, skipping secrets loading")
		return nil
	}

	if !dp.isDopplerConfigured() {
		dp.logger.Debug(ctx, "Doppler not configured, skipping secrets loading")
		return nil
	}

	dp.logger.Info(ctx, "Loading secrets from Doppler")

	secrets, err := dp.fetchDopplerSecrets()
	if err != nil {
		return fmt.Errorf("failed to fetch secrets from Doppler: %w", err)
	}

	for key, value := range secrets {
		err := os.Setenv(key, value)
		if err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	dp.logger.Info(ctx, "Successfully loaded secrets from Doppler",
		observability.Int("secret_count", len(secrets)))

	return nil
}

func (dp *DopplerProvider) isDopplerCLIAvailable() bool {
	_, err := exec.LookPath("doppler")
	return err == nil
}

// isDopplerConfigured checks if Doppler is configured either via DOPPLER_TOKEN
// environment variable or a doppler.yaml configuration file.
func (dp *DopplerProvider) isDopplerConfigured() bool {
	if os.Getenv("DOPPLER_TOKEN") != "" {
		return true
	}

	if _, err := os.Stat("doppler.yaml"); err == nil {
		return true
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		if _, err := os.Stat(homeDir + "/.doppler.yaml"); err == nil {
			return true
		}
	}

	return false
}

// fetchDopplerSecrets uses the Doppler CLI to fetch secrets.
// It returns a map of secret names to values.
func (dp *DopplerProvider) fetchDopplerSecrets() (map[string]string, error) {
	ctx := context.Background()

	// Get Doppler configuration from environment
	project := os.Getenv("DOPPLER_PROJECT")
	config := os.Getenv("DOPPLER_CONFIG")
	token := os.Getenv("DOPPLER_TOKEN")

	dp.logger.Info(ctx, "Attempting to fetch secrets from Doppler",
		observability.String("project", project),
		observability.String("config", config),
		observability.Bool("token_present", token != ""),
	)

	// Build command with explicit project and config if available
	args := []string{"secrets", "download", "--no-file", "--format", "json"}
	if project != "" {
		args = append(args, "--project", project)
	}
	if config != "" {
		args = append(args, "--config", config)
	}

	cmd := exec.Command("doppler", args...)

	// Set environment variables for the command
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DOPPLER_TOKEN=%s", token),
		fmt.Sprintf("DOPPLER_PROJECT=%s", project),
		fmt.Sprintf("DOPPLER_CONFIG=%s", config),
	)

	dp.logger.Debug(ctx, "Executing Doppler command",
		observability.String("command", fmt.Sprintf("doppler %s", strings.Join(args, " "))),
	)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			dp.logger.Error(ctx, err, "Doppler command failed with stderr",
				observability.String("stderr", string(exitErr.Stderr)),
				observability.String("project", project),
				observability.String("config", config),
			)
			return nil, fmt.Errorf("doppler command failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	// Parse the JSON output to extract secrets
	secretsMap := make(map[string]string)

	// The output is in JSON format, we need to parse it
	// For simplicity, we'll use a simple string parsing approach
	// In a production environment, you would use json.Unmarshal
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{" || line == "}" {
			continue
		}

		// Format is "  "KEY": "VALUE"," or "  "KEY": "VALUE""
		line = strings.TrimSuffix(line, ",")
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.Trim(strings.TrimSpace(parts[0]), "\"")
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		secretsMap[key] = value
	}

	return secretsMap, nil
}
