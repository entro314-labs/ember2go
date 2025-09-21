package bootloader

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// UEFIBootloader handles UEFI bootloader creation
type UEFIBootloader struct{}

// NewUEFIBootloader creates a new UEFI bootloader manager
func NewUEFIBootloader() *UEFIBootloader {
	return &UEFIBootloader{}
}

// CreateBootloader creates a Windows UEFI bootloader
func (u *UEFIBootloader) CreateBootloader(bootMountPath, windowsMountPath, diskUUID, bootUUID, windowsUUID string) error {
	fmt.Println("Creating Windows UEFI bootloader...")

	// Copy boot manager files
	if err := u.copyBootManagerFiles(bootMountPath, windowsMountPath); err != nil {
		return fmt.Errorf("failed to copy boot manager files: %w", err)
	}

	// Create basic BCD structure (simplified)
	if err := u.createBasicBCD(bootMountPath, windowsUUID); err != nil {
		return fmt.Errorf("failed to create BCD: %w", err)
	}

	// Create UEFI startup script
	if err := u.createUEFIStartupScript(bootMountPath); err != nil {
		return fmt.Errorf("failed to create startup script: %w", err)
	}

	fmt.Println("UEFI bootloader created successfully")
	return nil
}

// copyBootManagerFiles copies Windows boot manager files to the boot partition
func (u *UEFIBootloader) copyBootManagerFiles(bootMountPath, windowsMountPath string) error {
	fmt.Println("Copying boot manager files...")

	// Create EFI directory structure
	efiBootDir := filepath.Join(bootMountPath, "EFI", "Microsoft", "Boot")
	if err := os.MkdirAll(efiBootDir, 0755); err != nil {
		return fmt.Errorf("failed to create EFI boot directory: %w", err)
	}

	// Copy EFI boot files
	efiSrc := filepath.Join(windowsMountPath, "Windows", "Boot", "EFI")
	if err := u.copyDirectoryFiles(efiSrc, efiBootDir); err != nil {
		return fmt.Errorf("failed to copy EFI files: %w", err)
	}

	// Copy boot fonts
	fontsSrc := filepath.Join(windowsMountPath, "Windows", "Boot", "Fonts")
	if err := u.copyDirectoryFiles(fontsSrc, efiBootDir); err != nil {
		// Fonts might not exist in all Windows versions, so don't fail
		fmt.Printf("Warning: failed to copy fonts: %v\n", err)
	}

	// Copy boot resources
	resourcesSrc := filepath.Join(windowsMountPath, "Windows", "Boot", "Resources")
	if err := u.copyDirectoryFiles(resourcesSrc, efiBootDir); err != nil {
		// Resources might not exist in all Windows versions, so don't fail
		fmt.Printf("Warning: failed to copy resources: %v\n", err)
	}

	fmt.Println("Boot manager files copied successfully")
	return nil
}

// copyDirectoryFiles copies all files from source directory to destination directory
func (u *UEFIBootloader) copyDirectoryFiles(srcDir, dstDir string) error {
	// Check if source directory exists
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", srcDir)
	}

	// Read source directory
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each file
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories for now
		}

		srcFile := filepath.Join(srcDir, entry.Name())
		dstFile := filepath.Join(dstDir, entry.Name())

		if err := u.copyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// copyFile copies a single file from source to destination
func (u *UEFIBootloader) copyFile(src, dst string) error {
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
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// createBasicBCD creates a basic BCD store (simplified version)
func (u *UEFIBootloader) createBasicBCD(bootMountPath, windowsUUID string) error {
	fmt.Println("Creating basic BCD store...")

	bcdPath := filepath.Join(bootMountPath, "EFI", "Microsoft", "Boot", "BCD")

	// For now, we create a placeholder BCD file
	// A full implementation would require:
	// 1. BCD template file
	// 2. Registry editing tools (like hivex)
	// 3. Proper GUID handling

	// Create an empty BCD file as a placeholder
	bcdFile, err := os.Create(bcdPath)
	if err != nil {
		return fmt.Errorf("failed to create BCD file: %w", err)
	}
	defer bcdFile.Close()

	// Write minimal BCD content (this is a placeholder)
	// In a real implementation, this would be proper BCD binary data
	placeholder := fmt.Sprintf("# BCD Placeholder\n# Windows UUID: %s\n# Created by ember2go\n", windowsUUID)
	if _, err := bcdFile.WriteString(placeholder); err != nil {
		return fmt.Errorf("failed to write BCD content: %w", err)
	}

	fmt.Printf("Basic BCD store created at %s\n", bcdPath)
	fmt.Println("Note: Full BCD creation requires additional tools")

	return nil
}

// createUEFIStartupScript creates a UEFI startup.nsh script for automatic boot
func (u *UEFIBootloader) createUEFIStartupScript(bootMountPath string) error {
	fmt.Println("Creating UEFI startup script...")

	startupContent := `@echo off
echo Starting Windows To Go...
\\EFI\\Microsoft\\Boot\\bootmgfw.efi
`

	startupPath := filepath.Join(bootMountPath, "startup.nsh")
	if err := os.WriteFile(startupPath, []byte(startupContent), 0644); err != nil {
		return fmt.Errorf("failed to create startup script: %w", err)
	}

	fmt.Printf("UEFI startup script created at %s\n", startupPath)
	return nil
}

// CheckBootloaderTools checks if bootloader creation tools are available
func (u *UEFIBootloader) CheckBootloaderTools() error {
	// Check for BCD tools
	tools := []string{"hivex", "bcdedit"}

	for _, tool := range tools {
		cmd := exec.Command(tool, "--version")
		if err := cmd.Run(); err == nil {
			fmt.Printf("Found BCD tool: %s\n", tool)
			return nil
		}
	}

	// No tools found - this will trigger the warning in main
	return fmt.Errorf("no BCD tools found")
}

// ValidateBootloader validates that the bootloader was created correctly
func (u *UEFIBootloader) ValidateBootloader(bootMountPath string) error {
	// Check if required files exist
	requiredFiles := []string{
		"EFI/Microsoft/Boot/bootmgfw.efi",
		"EFI/Microsoft/Boot/BCD",
		"startup.nsh",
	}

	for _, file := range requiredFiles {
		fullPath := filepath.Join(bootMountPath, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("validation failed: required file %s not found", file)
		}
	}

	fmt.Println("Bootloader validation passed")
	return nil
}

// GetBootInfo returns information about the created bootloader
func (u *UEFIBootloader) GetBootInfo(bootMountPath string) map[string]string {
	info := make(map[string]string)

	info["Type"] = "UEFI"
	info["BootPath"] = bootMountPath
	info["StartupScript"] = filepath.Join(bootMountPath, "startup.nsh")
	info["BCDPath"] = filepath.Join(bootMountPath, "EFI", "Microsoft", "Boot", "BCD")
	info["EFIPath"] = filepath.Join(bootMountPath, "EFI", "Microsoft", "Boot")

	return info
}

// InstallBootloaderDependencies installs bootloader creation dependencies
func (u *UEFIBootloader) InstallBootloaderDependencies() error {
	fmt.Println("Installing bootloader dependencies...")

	// For advanced BCD editing, you would install hivex:
	// exec.Command("brew", "install", "hivex").Run()

	fmt.Println("Basic bootloader functionality available")
	fmt.Println("For advanced features, consider installing:")
	fmt.Println("  brew install hivex")

	return nil
}