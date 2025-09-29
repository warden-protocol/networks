package main

import (
	"context"
	"dagger/ci/internal/dagger"
	"fmt"
	"strings"
)

type Ci struct{}

// ValidateGentxCli validates Genesis Transaction files using the check-genesis tool (CLI-friendly)
// This function returns a simple string output and uses appropriate exit codes for CI/CD
func (m *Ci) ValidateGentxCli(
	ctx context.Context,
	// Source directory containing the repository
	source *dagger.Directory,
	// Network to validate (default: mainnet)
	// +optional
	// +default="mainnet"
	network string,
	// Wardend version to use for validation
	// +optional
	// +default="v0.7.0"
	wardendVersion string,
	// Go version for building check-genesis tool
	// +optional
	// +default="1.24"
	goVersion string,
) (string, error) {
	result, err := m.ValidateGentx(ctx, source, network, wardendVersion, goVersion)
	if err != nil {
		return "", err
	}

	// Format output and return error if validation failed
	output := fmt.Sprintf("Status: %s, Network: %s, Files: %d, Message: %s",
		result.Status, result.NetworkValidated, result.FilesValidated, result.Message)

	if result.Status == "failed" {
		return output, fmt.Errorf("validation failed")
	}

	return output, nil
}

// ValidateGentx validates Genesis Transaction files using the check-genesis tool
func (m *Ci) ValidateGentx(
	ctx context.Context,
	// Source directory containing the repository
	source *dagger.Directory,
	// Network to validate (default: mainnet)
	// +optional
	// +default="mainnet"
	network string,
	// Wardend version to use for validation
	// +optional
	// +default="v0.7.0"
	wardendVersion string,
	// Go version for building check-genesis tool
	// +optional
	// +default="1.24"
	goVersion string,
) (*ValidationResult, error) {
	// Create a container with Go for building the check-genesis tool
	goContainer := dag.Container().
		From(fmt.Sprintf("golang:%s", goVersion)).
		WithWorkdir("/workspace").
		WithDirectory("/workspace", source)

	// Build the check-genesis tool
	checkGenesis := goContainer.
		WithWorkdir("/workspace/utils/check-genesis").
		WithExec([]string{"go", "mod", "tidy"}).
		WithExec([]string{"go", "build", "-o", "check-genesis", "."}).
		File("check-genesis")

	// Create validation container with wardend
	validationContainer := dag.Container().
		From(fmt.Sprintf("ghcr.io/warden-protocol/wardenprotocol/wardend:%s", wardendVersion)).
		WithUser("root").
		WithWorkdir("/validation").
		WithDirectory("/validation/source", source).
		WithFile("/validation/check-genesis", checkGenesis)

	// Get the list of gentx files to validate
	gentxFiles, err := m.getGentxFiles(ctx, source, network)
	if err != nil {
		return nil, fmt.Errorf("failed to get gentx files: %w", err)
	}

	if len(gentxFiles) == 0 {
		return &ValidationResult{
			Status:           "no-files",
			Message:          fmt.Sprintf("No %s GenTx files found to validate", network),
			FilesValidated:   0,
			NetworkValidated: network,
		}, nil
	}

	// Validate each gentx file
	var validationResults []FileValidationResult
	validatedCount := 0

	for _, gentxFile := range gentxFiles {
		result, err := m.validateSingleGentx(ctx, validationContainer, gentxFile, network)
		if err != nil {
			return nil, fmt.Errorf("validation setup failed for %s: %w", gentxFile, err)
		}

		validationResults = append(validationResults, result)
		if result.Status == "passed" {
			validatedCount++
		}
	}

	// Determine overall status
	status := "passed"
	failedFiles := []string{}
	for _, result := range validationResults {
		if result.Status == "failed" {
			status = "failed"
			failedFiles = append(failedFiles, result.File)
		}
	}

	message := fmt.Sprintf("Validated %d files", validatedCount)
	if status == "failed" {
		message = fmt.Sprintf("Validation failed. %d passed, %d failed",
			validatedCount, len(failedFiles))
	}

	return &ValidationResult{
		Status:           status,
		Message:          message,
		FilesValidated:   validatedCount,
		NetworkValidated: network,
		Results:          validationResults,
		FailedFiles:      failedFiles,
	}, nil
}

// ValidationResult represents the overall validation result
type ValidationResult struct {
	Status           string                 `json:"status"`
	Message          string                 `json:"message"`
	FilesValidated   int                    `json:"files_validated"`
	NetworkValidated string                 `json:"network_validated"`
	Results          []FileValidationResult `json:"results,omitempty"`
	FailedFiles      []string               `json:"failed_files,omitempty"`
}

// FileValidationResult represents the validation result for a single file
type FileValidationResult struct {
	File    string `json:"file"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// getGentxFiles returns the list of gentx files to validate
func (m *Ci) getGentxFiles(
	ctx context.Context,
	source *dagger.Directory,
	network string,
) ([]string, error) {
	if network != "mainnet" {
		return nil, fmt.Errorf("only mainnet network is supported")
	}

	// List files in the gentx directory
	gentxDir := fmt.Sprintf("%s/gentx", network)
	entries, err := source.Directory(gentxDir).Entries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list gentx directory: %w", err)
	}

	var gentxFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry, ".json") {
			gentxFiles = append(gentxFiles, fmt.Sprintf("%s/%s", gentxDir, entry))
		}
	}

	return gentxFiles, nil
}

// validateSingleGentx validates a single gentx file with detailed error capturing
func (m *Ci) validateSingleGentx(
	ctx context.Context,
	container *dagger.Container,
	gentxFile string,
	network string,
) (FileValidationResult, error) {
	// Setup the validation environment for this specific file
	genesisFile := fmt.Sprintf("%s/init_genesis.json", network)

	validationContainer := container.
		WithFile("/validation/init_genesis.json",
			container.Directory("/validation/source").File(genesisFile)).
		WithFile("/validation/gentx.json",
			container.Directory("/validation/source").File(gentxFile)).
		WithExec([]string{"mkdir", "-p", "/tmp/.warden/config/gentx"}).
		WithEnvVariable("NO_COLOR", "1").
		WithEnvVariable("HOME", "/tmp")

	// Run the validation command
	validationResult := validationContainer.
		WithWorkdir("/tmp").
		WithFile("./init_genesis.json", container.Directory("/validation/source").File(genesisFile)).
		WithExec([]string{"/validation/check-genesis", "/validation/gentx.json"})

	// Try to get stdout
	_, stdoutErr := validationResult.Stdout(ctx)
	stderr, _ := validationResult.Stderr(ctx)

	// If validation failed
	if stdoutErr != nil {
		errorMessage := "Validation failed"

		// Try to get stderr for error details
		if stderr != "" {
			errorMessage = fmt.Sprintf("Validation failed: %s", stderr)
		}

		// Try to get logs.txt for more detailed error information
		logResult := validationResult.WithExec([]string{"cat", "logs.txt"})
		if logs, logErr := logResult.Stdout(ctx); logErr == nil && logs != "" {
			// Extract the last few lines or first error for a more specific message
			logLines := strings.Split(logs, "\n")
			for i := len(logLines) - 1; i >= 0; i-- {
				line := strings.TrimSpace(logLines[i])
				if line != "" && (strings.Contains(strings.ToLower(line), "error") ||
					strings.Contains(strings.ToLower(line), "failed") ||
					strings.Contains(strings.ToLower(line), "panic")) {
					errorMessage = fmt.Sprintf("Validation failed: %s", line)
					break
				}
			}
		}

		return FileValidationResult{
			File:    gentxFile,
			Status:  "failed",
			Message: errorMessage,
		}, nil
	}

	// Successful validation
	return FileValidationResult{
		File:    gentxFile,
		Status:  "passed",
		Message: "Validation successful",
	}, nil
}

// RunLocalValidation runs validation locally with verbose output
func (m *Ci) RunLocalValidation(
	ctx context.Context,
	// Source directory containing the repository
	source *dagger.Directory,
	// Network to validate (default: mainnet)
	// +optional
	// +default="mainnet"
	network string,
	// Wardend version to use for validation
	// +optional
	// +default="v0.7.0"
	wardendVersion string,
	// Go version for building check-genesis tool
	// +optional
	// +default="1.24"
	goVersion string,
) (string, error) {
	result, err := m.ValidateGentx(ctx, source, network, wardendVersion, goVersion)
	if err != nil {
		return "", err
	}

	// Format the output for local development
	var output strings.Builder
	output.WriteString("🚀 GenTx Validation Results\n")
	output.WriteString("===========================\n\n")
	output.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
	output.WriteString(fmt.Sprintf("Network: %s\n", result.NetworkValidated))
	output.WriteString(fmt.Sprintf("Files Validated: %d\n", result.FilesValidated))
	output.WriteString(fmt.Sprintf("Message: %s\n\n", result.Message))

	if len(result.Results) > 0 {
		output.WriteString("Individual Results:\n")
		for _, fileResult := range result.Results {
			status := "✅"
			if fileResult.Status == "failed" {
				status = "❌"
			}
			output.WriteString(fmt.Sprintf("  %s %s", status, fileResult.File))
			if fileResult.Message != "" {
				output.WriteString(fmt.Sprintf(" - %s", fileResult.Message))
			}
			output.WriteString("\n")
		}
	}

	if len(result.FailedFiles) > 0 {
		output.WriteString("\nFailed Files:\n")
		for _, file := range result.FailedFiles {
			output.WriteString(fmt.Sprintf("  - %s\n", file))
		}
	}

	return output.String(), nil
}

// CopyAllGentx copies all gentx files to the validation environment
func (m *Ci) CopyAllGentx(
	ctx context.Context,
	// Source directory containing the repository
	source *dagger.Directory,
	// Network to copy from (default: mainnet)
	// +optional
	// +default="mainnet"
	network string,
	// Wardend version to use for validation
	// +optional
	// +default="v0.7.0"
	wardendVersion string,
) (*dagger.Container, error) {
	// Get the list of gentx files
	gentxFiles, err := m.getGentxFiles(ctx, source, network)
	if err != nil {
		return nil, fmt.Errorf("failed to get gentx files: %w", err)
	}

	if len(gentxFiles) == 0 {
		return nil, fmt.Errorf("no %s GenTx files found to copy", network)
	}

	// Create validation container with wardend
	container := dag.Container().
		From(fmt.Sprintf("ghcr.io/warden-protocol/wardenprotocol/wardend:%s", wardendVersion)).
		WithUser("root").
		WithWorkdir("/validation").
		WithDirectory("/validation/source", source).
		WithExec([]string{"mkdir", "-p", "/tmp/.warden/config/gentx"}).
		WithEnvVariable("NO_COLOR", "1").
		WithEnvVariable("HOME", "/tmp")

	// Copy all gentx files to the gentx directory
	for _, gentxFile := range gentxFiles {
		// Extract filename from path
		parts := strings.Split(gentxFile, "/")
		filename := parts[len(parts)-1]

		// Copy each gentx file to the gentx directory with its original name
		container = container.WithFile(
			fmt.Sprintf("/tmp/.warden/config/gentx/%s", filename),
			container.Directory("/validation/source").File(gentxFile),
		)
	}

	// Also copy the genesis file
	genesisFile := fmt.Sprintf("%s/init_genesis.json", network)
	container = container.WithFile("/tmp/init_genesis.json",
		container.Directory("/validation/source").File(genesisFile))

	return container, nil
}

// ValidateAllGentxTogether validates all gentx files together in one validation run
func (m *Ci) ValidateAllGentxTogether(
	ctx context.Context,
	// Source directory containing the repository
	source *dagger.Directory,
	// Network to validate (default: mainnet)
	// +optional
	// +default="mainnet"
	network string,
	// Wardend version to use for validation
	// +optional
	// +default="v0.7.0"
	wardendVersion string,
	// Go version for building check-genesis tool
	// +optional
	// +default="1.24"
	goVersion string,
) (string, error) {
	// Build the check-genesis tool
	goContainer := dag.Container().
		From(fmt.Sprintf("golang:%s", goVersion)).
		WithWorkdir("/workspace").
		WithDirectory("/workspace", source)

	checkGenesis := goContainer.
		WithWorkdir("/workspace/utils/check-genesis").
		WithExec([]string{"go", "mod", "tidy"}).
		WithExec([]string{"go", "build", "-o", "check-genesis", "."}).
		File("check-genesis")

	// Create validation container with wardend
	validationContainer := dag.Container().
		From(fmt.Sprintf("ghcr.io/warden-protocol/wardenprotocol/wardend:%s", wardendVersion)).
		WithUser("root").
		WithWorkdir("/validation").
		WithDirectory("/validation/source", source).
		WithFile("/validation/check-genesis", checkGenesis).
		WithEnvVariable("NO_COLOR", "1").
		WithEnvVariable("HOME", "/tmp")

	// Copy the genesis file
	genesisFile := fmt.Sprintf("%s/init_genesis.json", network)
	validationContainer = validationContainer.WithFile("/validation/init_genesis.json",
		validationContainer.Directory("/validation/source").File(genesisFile))

	// Run the check-genesis tool on the entire gentx directory
	gentxDir := fmt.Sprintf("/validation/source/%s/gentx", network)
	validationResult := validationContainer.
		WithWorkdir("/validation").
		WithExec([]string{"/validation/check-genesis", gentxDir})

	// Get the result
	stdout, stdoutErr := validationResult.Stdout(ctx)
	stderr, _ := validationResult.Stderr(ctx)

	if stdoutErr != nil {
		// Try to get additional debug information from logs
		debugResult := validationResult.WithExec([]string{"cat", "logs.txt"})
		debugLogs, _ := debugResult.Stdout(ctx)

		return fmt.Sprintf(
			"❌ Validation FAILED:\n%s\n\nStderr:\n%s\n\nDebug logs:\n%s",
			stdout,
			stderr,
			debugLogs,
		), stdoutErr
	}

	return fmt.Sprintf("✅ Validation PASSED:\n%s", stdout), nil
}

// TestCheckGenesisTool tests that the check-genesis tool can be built and run
func (m *Ci) TestCheckGenesisTool(
	ctx context.Context,
	// Source directory containing the repository
	source *dagger.Directory,
	// Go version for building check-genesis tool
	// +optional
	// +default="1.24"
	goVersion string,
) (string, error) {
	// Create a container with Go for building the check-genesis tool
	container := dag.Container().
		From(fmt.Sprintf("golang:%s", goVersion)).
		WithWorkdir("/workspace").
		WithDirectory("/workspace", source).
		WithWorkdir("/workspace/utils/check-genesis")

	// Test building the tool
	buildContainer := container.
		WithExec([]string{"go", "mod", "tidy"}).
		WithExec([]string{"go", "build", "-o", "check-genesis", "."})

	// Test that the binary exists and is executable
	lsContainer := buildContainer.
		WithExec([]string{"ls", "-la", "check-genesis"})

	lsOutput, err := lsContainer.Stdout(ctx)
	if err != nil {
		return fmt.Sprintf("Build test failed - binary not found:\n%s", lsOutput), err
	}

	// Test running the binary (will show usage and exit with code 1, which is expected)
	usageContainer := buildContainer.
		WithExec([]string{"./check-genesis"})

	// Try to get stdout even if exit code is non-zero
	usageOutput, _ := usageContainer.Stdout(ctx)
	usageError, _ := usageContainer.Stderr(ctx)

	result := fmt.Sprintf(
		"✅ check-genesis tool build and test successful:\n\nFile details:\n%s\nUsage output:\n%s",
		lsOutput,
		usageOutput,
	)
	if usageError != "" {
		result += fmt.Sprintf("\nUsage stderr:\n%s", usageError)
	}

	return result, nil
}
