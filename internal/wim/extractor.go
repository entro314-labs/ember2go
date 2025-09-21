package wim

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Extractor handles WIM file extraction operations
type Extractor struct{}

// NewExtractor creates a new WIM extractor
func NewExtractor() *Extractor {
	return &Extractor{}
}

// ExtractWindowsEdition extracts a Windows edition to the target path
func (e *Extractor) ExtractWindowsEdition(ctx context.Context, wimInfo *WIMInfo, edition *WindowsEdition, targetPath string, options *ExtractionOptions) error {
	if options == nil {
		options = &ExtractionOptions{
			NoACLs:              true,
			NoAttributes:        true,
			IncludeInvalidNames: true,
		}
	}

	// Build command arguments
	args := []string{"apply", wimInfo.Path, strconv.Itoa(edition.Index), targetPath}

	if options.NoACLs {
		args = append(args, "--no-acls")
	}
	if options.NoAttributes {
		args = append(args, "--no-attributes")
	}
	if options.IncludeInvalidNames {
		args = append(args, "--include-invalid-names")
	}

	cmd := exec.CommandContext(ctx, "wimlib-imagex", args...)

	// Set up progress monitoring if callback is provided
	if options.ProgressCallback != nil {
		return e.extractWithProgress(cmd, options.ProgressCallback)
	}

	// Simple extraction without progress
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract Windows edition: %w", err)
	}

	return nil
}

// extractWithProgress runs the extraction command and monitors progress
func (e *Extractor) extractWithProgress(cmd *exec.Cmd, callback ProgressCallback) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start extraction: %w", err)
	}

	// Monitor stdout for progress information
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			e.parseProgressLine(line, callback)
		}
	}()

	// Monitor stderr for errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// Log error lines or parse them for useful information
			if callback != nil {
				callback(0, 0, "Error: "+line)
			}
		}
	}()

	return cmd.Wait()
}

// parseProgressLine parses a progress line from wimlib-imagex output
func (e *Extractor) parseProgressLine(line string, callback ProgressCallback) {
	if callback == nil {
		return
	}

	// wimlib-imagex typically outputs progress in various formats
	// Look for percentage indicators or file counts
	line = strings.TrimSpace(line)

	// Look for percentage patterns like "25.0% complete"
	if strings.Contains(line, "% complete") || strings.Contains(line, "%") {
		callback(0, 100, line)
		return
	}

	// Look for file extraction patterns
	if strings.Contains(line, "Extracting") || strings.Contains(line, "Applying") {
		callback(0, 0, line)
		return
	}

	// Generic progress message
	if line != "" {
		callback(0, 0, line)
	}
}

// ValidateExtraction validates that the extraction was successful
func (e *Extractor) ValidateExtraction(targetPath string, edition *WindowsEdition) error {
	// Check if basic Windows directories exist
	requiredPaths := []string{
		"Windows",
		"Windows/System32",
		"Program Files",
	}

	for _, path := range requiredPaths {
		fullPath := targetPath + "/" + path
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("validation failed: required path %s not found", path)
		}
	}

	return nil
}

// GetExtractionProgress creates a progress callback that logs to stdout
func (e *Extractor) GetExtractionProgress() ProgressCallback {
	lastUpdate := time.Now()
	return func(current, total int64, message string) {
		now := time.Now()
		// Throttle updates to avoid spam
		if now.Sub(lastUpdate) < 500*time.Millisecond && message != "" {
			return
		}
		lastUpdate = now

		if total > 0 {
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%% - %s", percent, message)
		} else {
			fmt.Printf("\r%s", message)
		}

		// Clear the line if message is empty (final cleanup)
		if message == "" {
			fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
		}
	}
}

// EstimateExtractionTime estimates how long extraction might take
func (e *Extractor) EstimateExtractionTime(edition *WindowsEdition) time.Duration {
	// Very rough estimation based on file count and size
	// Assumes ~50MB/s extraction speed on average hardware
	bytesPerSecond := int64(50 * 1024 * 1024) // 50 MB/s

	if edition.TotalBytes > 0 {
		seconds := edition.TotalBytes / bytesPerSecond
		return time.Duration(seconds) * time.Second
	}

	// Fallback based on file count (very rough)
	if edition.FileCount > 0 {
		// Assume 1000 files per second
		seconds := edition.FileCount / 1000
		return time.Duration(seconds) * time.Second
	}

	// Default estimate for unknown sizes
	return 20 * time.Minute
}

// GetSupportedFormats returns the WIM formats supported by wimlib
func (e *Extractor) GetSupportedFormats() ([]string, error) {
	cmd := exec.Command("wimlib-imagex", "--help")
	_, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get wimlib help: %w", err)
	}

	// Parse supported formats from help output
	// This is a simplified version - actual implementation would parse the help text
	return []string{"WIM", "ESD", "SWM"}, nil
}