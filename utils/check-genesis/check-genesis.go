package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// Emoji constants for better visual appeal
const (
	EmojiCheck   = "✅"
	EmojiError   = "❌"
	EmojiWarning = "⚠️"
	EmojiInfo    = "ℹ️"
	EmojiRocket  = "🚀"
	EmojiMoney   = "💰"
	EmojiFile    = "📄"
	EmojiFolder  = "📁"
	EmojiGear    = "⚙️"
	EmojiClock   = "⏱️"
	EmojiTarget  = "🎯"
)

const (
	// Default configuration
	WARDEND             = "wardend"
	WARDDIR             = ".warden"
	NETWORK             = "barra_9191-1"
	TIMEOUT             = 60 // seconds
	REQ_FEE             = "180000000000000000"
	LOGS_FILE           = "logs.txt"
	INIT_GENESIS        = "./init_genesis.json"
	PANIC_WAIT_TIME     = 5  // seconds to wait after panic detection for additional logs
	PANIC_CONTEXT_LINES = 50 // number of lines to show after panic line
)

// GentxFee represents the fee structure in a gentx file
type GentxFee struct {
	Amount []struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"amount"`
}

// GentxAuthInfo represents the auth_info structure in a gentx file
type GentxAuthInfo struct {
	Fee GentxFee `json:"fee"`
}

// Gentx represents the structure of a genesis transaction file
type Gentx struct {
	AuthInfo GentxAuthInfo `json:"auth_info"`
}

// Logger provides colored and formatted output
type Logger struct {
	useColors bool
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	// Check if we should use colors (disabled in non-TTY environments like CI)
	useColors := isTerminal() && os.Getenv("NO_COLOR") == ""
	return &Logger{useColors: useColors}
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Colorize applies color to text if colors are enabled
func (l *Logger) colorize(color, text string) string {
	if !l.useColors {
		return text
	}
	return color + text + ColorReset
}

// Header prints a colored header with border
func (l *Logger) header(text string) {
	border := strings.Repeat("=", len(text)+4)
	fmt.Printf("\n%s\n", l.colorize(ColorCyan+ColorBold, border))
	fmt.Printf(
		"%s %s %s\n",
		l.colorize(ColorCyan+ColorBold, "🎯"),
		l.colorize(ColorWhite+ColorBold, text),
		l.colorize(ColorCyan+ColorBold, "🎯"),
	)
	fmt.Printf("%s\n\n", l.colorize(ColorCyan+ColorBold, border))
}

// Info prints an info message
func (l *Logger) info(emoji, message string) {
	fmt.Printf(
		"%s %s %s\n",
		l.colorize(ColorBlue+ColorBold, emoji),
		l.colorize(ColorWhite, message),
		l.colorize(ColorDim, getTimestamp()),
	)
}

// Success prints a success message
func (l *Logger) success(emoji, message string) {
	fmt.Printf(
		"%s %s %s\n",
		l.colorize(ColorGreen+ColorBold, emoji),
		l.colorize(ColorGreen, message),
		l.colorize(ColorDim, getTimestamp()),
	)
}

// Warning prints a warning message
func (l *Logger) warning(emoji, message string) {
	fmt.Printf(
		"%s %s %s\n",
		l.colorize(ColorYellow+ColorBold, emoji),
		l.colorize(ColorYellow, message),
		l.colorize(ColorDim, getTimestamp()),
	)
}

// Error prints an error message
func (l *Logger) error(emoji, message string) {
	fmt.Printf(
		"%s %s %s\n",
		l.colorize(ColorRed+ColorBold, emoji),
		l.colorize(ColorRed, message),
		l.colorize(ColorDim, getTimestamp()),
	)
}

// Step prints a step with progress indicator
func (l *Logger) step(stepNum, totalSteps int, emoji, message string) {
	progress := fmt.Sprintf("[%d/%d]", stepNum, totalSteps)
	fmt.Printf("%s %s %s %s\n",
		l.colorize(ColorPurple+ColorBold, progress),
		l.colorize(ColorBlue+ColorBold, emoji),
		l.colorize(ColorWhite, message),
		l.colorize(ColorDim, getTimestamp()))
}

// Detail prints detailed information with indentation
func (l *Logger) detail(message string) {
	fmt.Printf("    %s %s\n", l.colorize(ColorCyan, "→"), l.colorize(ColorDim, message))
}

// Progress prints a progress indicator
func (l *Logger) progress(message string, duration time.Duration) {
	fmt.Printf("    %s %s %s\n",
		l.colorize(ColorYellow, EmojiClock),
		l.colorize(ColorDim, message),
		l.colorize(ColorDim, fmt.Sprintf("(%.1fs)", duration.Seconds())))
}

// getTimestamp returns a formatted timestamp
func getTimestamp() string {
	return fmt.Sprintf("[%s]", time.Now().Format("15:04:05"))
}

func main() {
	logger := NewLogger()

	if len(os.Args) < 2 {
		logger.error(EmojiError, "Usage: go run check-genesis.go <gentx-file-or-directory>")
		logger.error(EmojiError, "  Examples:")
		logger.error(EmojiError, "    go run check-genesis.go gentx-validator1.json")
		logger.error(EmojiError, "    go run check-genesis.go /path/to/gentx/directory")
		os.Exit(1)
	}

	gentxPath := os.Args[1]

	logger.header("WARDEN GENESIS TRANSACTION VALIDATOR")

	// Determine if the path is a file or directory
	gentxFiles, err := getGentxFiles(gentxPath)
	if err != nil {
		logger.error(EmojiError, fmt.Sprintf("Failed to get gentx files: %v", err))
		os.Exit(1)
	}

	if len(gentxFiles) == 0 {
		logger.error(EmojiError, "No gentx files found to validate")
		os.Exit(1)
	}

	logger.info(EmojiInfo, fmt.Sprintf("Found %d gentx file(s) to validate:", len(gentxFiles)))
	for i, file := range gentxFiles {
		logger.detail(fmt.Sprintf("%d. %s", i+1, file))
	}

	if err := checkGenesis(gentxFiles, logger); err != nil {
		logger.error(EmojiError, fmt.Sprintf("Validation failed: %v", err))
		os.Exit(1)
	}

	logger.success(EmojiCheck, "All gentx validation completed successfully!")
	logger.header("VALIDATION COMPLETE")
}

// getGentxFiles returns a list of gentx files from a file or directory path
func getGentxFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	var files []string

	if info.IsDir() {
		// If it's a directory, find all .json files in it
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				fullPath := filepath.Join(path, entry.Name())
				files = append(files, fullPath)
			}
		}
	} else {
		// If it's a file, just return that file
		if strings.HasSuffix(path, ".json") {
			files = append(files, path)
		} else {
			return nil, fmt.Errorf("file must have .json extension: %s", path)
		}
	}

	return files, nil
}

func checkGenesis(gentxFiles []string, logger *Logger) error {
	startTime := time.Now()

	// Setup directories
	logger.step(1, 9, EmojiFolder, "Setting up directories")
	if err := setupDirectories(logger); err != nil {
		return fmt.Errorf("failed to setup directories: %w", err)
	}

	// Update client.toml with correct chain-id
	logger.step(2, 9, EmojiGear, "Updating client configuration")
	if err := updateClientConfig(logger); err != nil {
		return fmt.Errorf("failed to update client config: %w", err)
	}

	// Copy initial genesis
	logger.step(3, 9, EmojiFile, "Copying initial genesis")
	if err := copyInitialGenesis(logger); err != nil {
		return fmt.Errorf("failed to copy initial genesis: %w", err)
	}

	// Validate all gentx fees first
	logger.step(4, 9, EmojiMoney, "Validating gentx fees")
	for i, gentxFile := range gentxFiles {
		logger.detail(
			fmt.Sprintf(
				"Validating fee for file %d/%d: %s",
				i+1,
				len(gentxFiles),
				filepath.Base(gentxFile),
			),
		)
		if err := validateGentxFee(gentxFile, logger); err != nil {
			return fmt.Errorf("gentx fee validation failed for %s: %w", gentxFile, err)
		}
	}
	logger.detail("All gentx fees validated successfully")

	// Copy all gentx files to the correct location
	logger.step(5, 9, EmojiFile, "Copying all gentx files")
	for i, gentxFile := range gentxFiles {
		logger.detail(
			fmt.Sprintf("Copying file %d/%d: %s", i+1, len(gentxFiles), filepath.Base(gentxFile)),
		)
		if err := copyGentxFile(gentxFile, logger); err != nil {
			return fmt.Errorf("failed to copy gentx file %s: %w", gentxFile, err)
		}
	}
	logger.detail(fmt.Sprintf("Successfully copied %d gentx files", len(gentxFiles)))

	// Collect gentxs
	logger.step(6, 9, EmojiGear, "Collecting all gentxs")
	if err := collectGentxs(logger); err != nil {
		return fmt.Errorf("failed to collect gentxs: %w", err)
	}

	// Validate genesis
	logger.step(7, 9, EmojiTarget, "Validating genesis")
	if err := validateGenesis(logger); err != nil {
		return fmt.Errorf("genesis validation failed: %w", err)
	}

	// Start node and check for panics
	logger.step(8, 9, EmojiRocket, "Starting node and running tests")
	if err := startAndTestNode(logger); err != nil {
		return fmt.Errorf("node start test failed: %w", err)
	}

	// Print last lines of log
	logger.step(9, 9, EmojiInfo, "Generating final report")
	if err := printLogTail(logger); err != nil {
		logger.warning(EmojiWarning, fmt.Sprintf("Failed to print log tail: %v", err))
	}

	duration := time.Since(startTime)
	logger.progress("Total validation time", duration)

	// Print summary
	logger.info(EmojiTarget, fmt.Sprintf("Successfully processed %d gentx files:", len(gentxFiles)))
	for i, file := range gentxFiles {
		logger.detail(fmt.Sprintf("✅ %d. %s", i+1, filepath.Base(file)))
	}

	return nil
}

func setupDirectories(logger *Logger) error {
	gentxDir := filepath.Join(WARDDIR, "config", "gentx")
	logger.detail(fmt.Sprintf("Creating directory: %s", gentxDir))
	return os.MkdirAll(gentxDir, 0755)
}

func copyInitialGenesis(logger *Logger) error {
	src := INIT_GENESIS
	dst := filepath.Join(WARDDIR, "config", "genesis.json")

	logger.detail(fmt.Sprintf("Copying %s → %s", src, dst))
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func updateClientConfig(logger *Logger) error {
	startTime := time.Now()
	clientConfigPath := filepath.Join(WARDDIR, "config", "client.toml")

	logger.detail(fmt.Sprintf("Updating client config: %s", clientConfigPath))

	// Check if client.toml exists
	if _, err := os.Stat(clientConfigPath); os.IsNotExist(err) {
		logger.detail("client.toml does not exist, will be created by wardend init if needed")
		// Run wardend init to create the default configuration if it doesn't exist
		cmd := exec.Command(WARDEND, "init", "temp-node", "--home", WARDDIR, "--chain-id", NETWORK)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to initialize wardend config: %w", err)
		}
		logger.detail("Initialized wardend configuration")
	}

	// Read the current client.toml file
	content, err := os.ReadFile(clientConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read client.toml: %w", err)
	}

	// Convert to string for processing
	configContent := string(content)

	// Update the chain-id line
	lines := strings.Split(configContent, "\n")
	updated := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "chain-id") && strings.Contains(trimmed, "=") {
			lines[i] = fmt.Sprintf(`chain-id = "%s"`, NETWORK)
			logger.detail(fmt.Sprintf("Updated chain-id to: %s", NETWORK))
			updated = true
			break
		}
	}

	// If chain-id line wasn't found, add it
	if !updated {
		// Look for a good place to insert it, preferably after a comment or at the beginning
		insertIndex := 0
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				insertIndex = i
				break
			}
		}

		newLine := fmt.Sprintf(`chain-id = "%s"`, NETWORK)
		lines = append(lines[:insertIndex], append([]string{newLine}, lines[insertIndex:]...)...)
		logger.detail(fmt.Sprintf("Added chain-id line: %s", NETWORK))
	}

	// Write the updated content back to the file
	updatedContent := strings.Join(lines, "\n")
	if err := os.WriteFile(clientConfigPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated client.toml: %w", err)
	}

	logger.detail("Successfully updated client.toml")
	logger.progress("Client config update completed", time.Since(startTime))
	return nil
}

func validateGentxFee(gentxFile string, logger *Logger) error {
	startTime := time.Now()

	// Read and parse the gentx file
	data, err := os.ReadFile(gentxFile)
	if err != nil {
		return fmt.Errorf("failed to read gentx file: %w", err)
	}

	var gentx Gentx
	if err := json.Unmarshal(data, &gentx); err != nil {
		return fmt.Errorf("failed to parse gentx JSON: %w", err)
	}

	// Check if fee amount exists
	if len(gentx.AuthInfo.Fee.Amount) == 0 {
		return fmt.Errorf("gentx fee is empty")
	}

	gentxFeeStr := gentx.AuthInfo.Fee.Amount[0].Amount
	if gentxFeeStr == "" {
		return fmt.Errorf("gentx fee amount is empty")
	}

	logger.detail(
		fmt.Sprintf("Found gentx fee: %s %s", gentxFeeStr, gentx.AuthInfo.Fee.Amount[0].Denom),
	)

	// Convert fee amounts to big.Int for comparison
	gentxFee := new(big.Int)
	if _, ok := gentxFee.SetString(gentxFeeStr, 10); !ok {
		return fmt.Errorf("invalid gentx fee format: %s", gentxFeeStr)
	}

	requiredFee := new(big.Int)
	if _, ok := requiredFee.SetString(REQ_FEE, 10); !ok {
		return fmt.Errorf("invalid required fee format: %s", REQ_FEE)
	}

	logger.detail(fmt.Sprintf("Required minimum fee: %s", REQ_FEE))

	// Compare fees
	if gentxFee.Cmp(requiredFee) < 0 {
		return fmt.Errorf(
			"gentx fee is less than minimum required fee: %s / %s",
			gentxFeeStr,
			REQ_FEE,
		)
	}

	logger.detail("Fee validation passed")
	logger.progress("Fee validation completed", time.Since(startTime))
	return nil
}

func copyGentxFile(gentxFile string, logger *Logger) error {
	dst := filepath.Join(WARDDIR, "config", "gentx", filepath.Base(gentxFile))
	logger.detail(fmt.Sprintf("Copying gentx file to: %s", dst))
	return copyFile(gentxFile, dst)
}

func collectGentxs(logger *Logger) error {
	startTime := time.Now()
	logger.detail("Running wardend genesis collect-gentxs...")

	cmd := exec.Command(WARDEND, "genesis", "collect-gentxs", "--home", WARDDIR)
	if err := runCommandWithLog(cmd, logger); err != nil {
		return err
	}

	logger.progress("Genesis collection completed", time.Since(startTime))
	return nil
}

func validateGenesis(logger *Logger) error {
	startTime := time.Now()
	logger.detail("Running wardend genesis validate-genesis...")

	cmd := exec.Command(WARDEND, "genesis", "validate-genesis", "--home", WARDDIR)
	if err := runCommandWithLog(cmd, logger); err != nil {
		return err
	}

	logger.progress("Genesis validation completed", time.Since(startTime))
	return nil
}

func runCommandWithLog(cmd *exec.Cmd, logger *Logger) error {
	// Log the command being executed
	logger.detail(fmt.Sprintf("Executing: %s", strings.Join(cmd.Args, " ")))

	// Open log file for appending
	logFile, err := os.OpenFile(LOGS_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// Write command header to log
	fmt.Fprintf(logFile, "\n=== Executing: %s ===\n", strings.Join(cmd.Args, " "))

	// Redirect stdout and stderr to log file
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Run the command and capture the exit code
	err = cmd.Run()

	// Write command result to log
	if err != nil {
		fmt.Fprintf(logFile, "=== Command failed with error: %v ===\n", err)
		logger.error(EmojiError, fmt.Sprintf("Command failed: %s", strings.Join(cmd.Args, " ")))

		// Try to get more details from the log
		if logErr := checkLogForFailure(logger); logErr != nil {
			return fmt.Errorf("command failed: %w, details: %v", err, logErr)
		}
		return fmt.Errorf("command failed: %w", err)
	} else {
		fmt.Fprintf(logFile, "=== Command completed successfully ===\n")
		logger.detail("Command completed successfully")
	}

	return nil
}

func checkLogForFailure(logger *Logger) error {
	file, err := os.Open(LOGS_FILE)
	if err != nil {
		return nil // If we can't read the log, don't fail
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var recentLines []string
	maxLines := 2 // Keep last 10 lines for context

	// Common error patterns to look for
	errorPatterns := []string{
		"error",
		"failed",
		"fail:",
		"panic:",
		"fatal",
		"invalid",
		"cannot",
		"unable to",
		"permission denied",
		"no such file",
		"connection refused",
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Keep a rolling buffer of recent lines
		recentLines = append(recentLines, line)
		if len(recentLines) > maxLines {
			recentLines = recentLines[1:]
		}

		// Check for error patterns
		lowerLine := strings.ToLower(line)
		for _, pattern := range errorPatterns {
			if strings.Contains(lowerLine, pattern) {
				// Return the problematic line with some context
				return fmt.Errorf("error detected in log: %s", line)
			}
		}
	}

	return scanner.Err()
}

func startAndTestNode(logger *Logger) error {
	startTime := time.Now()
	logger.detail("Starting wardend node in background...")

	// Start the node in background
	cmd := exec.Command(WARDEND, "start", "--home", WARDDIR)

	logFile, err := os.OpenFile(LOGS_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	logger.detail(fmt.Sprintf("Node started with PID: %d", cmd.Process.Pid))

	// Monitor for timeout and panics
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(TIMEOUT * time.Second)
	checkCount := 0

	for {
		select {
		case <-timeout:
			logger.info(EmojiClock, fmt.Sprintf("Timeout reached after %d seconds", TIMEOUT))
			// Kill the process
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				cmd.Process.Kill()
			}
			logger.progress("Node test completed", time.Since(startTime))
			return nil

		case <-ticker.C:
			checkCount++
			logger.detail(fmt.Sprintf("Health check %d/%d", checkCount, TIMEOUT/5))

			// Check for panics in log
			if err := checkLogForPanic(logger); err != nil {
				// Panic was detected and we already waited for additional logs
				logger.warning(
					EmojiWarning,
					"Terminating process after panic detection and log collection",
				)
				// Kill the process
				if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
					cmd.Process.Kill()
				}
				// Give it a moment to terminate gracefully
				time.Sleep(1 * time.Second)
				return err
			}

			// Check if process has exited
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				return fmt.Errorf("node process exited unexpectedly")
			}
		}
	}
}

func checkLogForPanic(logger *Logger) error {
	file, err := os.Open(LOGS_FILE)
	if err != nil {
		return nil // If we can't read the log, don't fail
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	panicLineIndex := -1

	// Read all current lines and check for panic
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if strings.Contains(line, "panic:") && panicLineIndex == -1 {
			panicLineIndex = len(lines) - 1 // Store the index of the first panic line
		}
	}

	if panicLineIndex != -1 {
		panicLine := lines[panicLineIndex]
		logger.error(EmojiError, fmt.Sprintf("Panic detected: %s", panicLine))
		logger.info(
			EmojiClock,
			fmt.Sprintf(
				"Waiting %d seconds for additional log output after panic...",
				PANIC_WAIT_TIME,
			),
		)

		// Wait for additional logs to be written
		time.Sleep(time.Duration(PANIC_WAIT_TIME) * time.Second)

		// Re-read the log file to capture any additional lines written after the panic
		file.Close()
		newFile, err := os.Open(LOGS_FILE)
		if err != nil {
			return fmt.Errorf("panic found in log: %s", panicLine)
		}
		defer newFile.Close()

		newScanner := bufio.NewScanner(newFile)
		var allLines []string
		for newScanner.Scan() {
			allLines = append(allLines, newScanner.Text())
		}

		// Show the panic line and at least 10 lines after it
		logger.error(
			EmojiError,
			fmt.Sprintf(
				"Panic context (showing panic line + %d lines after):",
				PANIC_CONTEXT_LINES,
			),
		)
		endIndex := panicLineIndex + PANIC_CONTEXT_LINES + 1
		if endIndex > len(allLines) {
			endIndex = len(allLines)
		}

		for i := panicLineIndex; i < endIndex; i++ {
			if i < len(allLines) {
				logger.detail(fmt.Sprintf("L%d: %s", i+1, allLines[i]))
			}
		}

		return fmt.Errorf("panic found in log: %s", panicLine)
	}

	return scanner.Err()
}

func printLogTail(logger *Logger) error {
	// Check if there was a panic in the logs to determine how many lines to show
	hasPanic := false
	file, err := os.Open(LOGS_FILE)
	if err != nil {
		return err
	}

	// Quick scan to see if there was a panic
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "panic:") {
			hasPanic = true
			break
		}
	}
	file.Close()

	// Determine how many lines to show
	linesToShow := 5
	if hasPanic {
		linesToShow = 15 // Show more lines if there was a panic
		logger.info(EmojiInfo, "Panic detected - showing last 15 lines from logs:")
	} else {
		logger.info(EmojiInfo, "Last 5 lines from logs:")
	}

	// Re-open file to read all lines
	file, err = os.Open(LOGS_FILE)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner = bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Print last N lines
	start := len(lines) - linesToShow
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		logger.detail(lines[i])
	}

	return scanner.Err()
}
