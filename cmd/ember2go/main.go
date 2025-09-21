package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/entro314-labs/ember2go/internal/bootloader"
	"github.com/entro314-labs/ember2go/internal/disk"
	"github.com/entro314-labs/ember2go/internal/tui"
	"github.com/entro314-labs/ember2go/internal/wim"
)

var (
	isoPath     string
	diskID      string
	editionIdx  int
	forceFlag   bool
	verboseFlag bool
	tuiFlag     bool

	// Build information (set by ldflags)
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
	builtBy   = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ember2go",
	Short: "Create portable Windows installations on USB devices",
	Long: `ember2go creates fully functional, portable Windows installations on USB devices.
These installations can boot on any UEFI-compatible computer.

Examples:
  ember2go list-disks                                    # List available USB drives
  ember2go list-editions /path/to/windows.iso           # Show Windows editions in ISO
  ember2go create --iso windows.iso --disk disk2        # Create Windows To Go
  ember2go create --iso windows.iso --disk disk2 --edition 2 --force`,
	Version: version,
}

var listDisksCmd = &cobra.Command{
	Use:   "list-disks",
	Short: "List available removable disks",
	Long:  "Display all removable USB drives that can be used for Windows To Go creation.",
	RunE:  runListDisks,
}

var listEditionsCmd = &cobra.Command{
	Use:   "list-editions <iso-path>",
	Short: "List Windows editions in an ISO file",
	Long:  "Analyze a Windows ISO file and display all available Windows editions.",
	Args:  cobra.ExactArgs(1),
	RunE:  runListEditions,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Windows To Go installation",
	Long: `Create a portable Windows installation on a USB device.
This will ERASE ALL DATA on the selected disk.

The process includes:
1. Formatting the USB drive with GPT partition table
2. Creating BOOT (FAT32) and WINDOWS (ExFAT) partitions
3. Extracting the selected Windows edition
4. Creating the UEFI bootloader`,
	RunE: runCreate,
}

var installDepsCmd = &cobra.Command{
	Use:   "install-deps",
	Short: "Install required dependencies",
	Long:  "Install wimlib and other tools required for Windows To Go creation.",
	RunE:  runInstallDeps,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show detailed version information",
	Long:  "Display version, build time, commit, and optimization details.",
	Run:   runVersion,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")

	// Create command flags
	createCmd.Flags().StringVar(&isoPath, "iso", "", "Path to Windows ISO file (required)")
	createCmd.Flags().StringVar(&diskID, "disk", "", "Target disk identifier, e.g. disk2 (only required for CLI mode)")
	createCmd.Flags().IntVar(&editionIdx, "edition", 1, "Windows edition index (default: 1)")
	createCmd.Flags().BoolVar(&forceFlag, "force", false, "Skip confirmation prompts")
	createCmd.Flags().BoolVar(&tuiFlag, "tui", true, "Use interactive TUI mode (default: true)")

	createCmd.MarkFlagRequired("iso")

	// Add subcommands
	rootCmd.AddCommand(listDisksCmd)
	rootCmd.AddCommand(listEditionsCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(installDepsCmd)
	rootCmd.AddCommand(versionCmd)
}

func runListDisks(cmd *cobra.Command, args []string) error {
	dm := disk.NewManager()

	// Check if diskutil is available
	if err := dm.CheckDiskutilAvailable(); err != nil {
		return fmt.Errorf("diskutil not available: %w", err)
	}

	disks, err := dm.GetRemovableDisks()
	if err != nil {
		return fmt.Errorf("failed to get removable disks: %w", err)
	}

	if len(disks) == 0 {
		fmt.Println("No removable disks found")
		return nil
	}

	fmt.Println("Available removable disks:")
	fmt.Println(strings.Repeat("-", 40))
	for _, d := range disks {
		fmt.Printf("  %s\n", d.String())
	}

	return nil
}

func runListEditions(cmd *cobra.Command, args []string) error {
	isoPath := args[0]

	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return fmt.Errorf("ISO file not found: %s", isoPath)
	}

	dm := disk.NewManager()
	parser := wim.NewParser()

	// Check dependencies
	if err := dm.CheckHdiutilAvailable(); err != nil {
		return fmt.Errorf("hdiutil not available: %w", err)
	}
	if err := parser.CheckWIMLibInstalled(); err != nil {
		return err
	}

	fmt.Printf("Analyzing ISO: %s\n", isoPath)

	// Mount ISO
	mountPath, err := dm.MountISO(isoPath)
	if err != nil {
		return fmt.Errorf("failed to mount ISO: %w", err)
	}
	defer dm.UnmountISO(mountPath)

	// Get WIM info
	wimInfo, err := parser.GetWIMInfo(mountPath)
	if err != nil {
		return fmt.Errorf("failed to analyze Windows editions: %w", err)
	}

	if len(wimInfo.Images) == 0 {
		fmt.Println("No Windows editions found in ISO")
		return nil
	}

	fmt.Println("\nAvailable Windows editions:")
	fmt.Println(strings.Repeat("-", 50))
	for _, edition := range wimInfo.Images {
		fmt.Printf("  %s\n", edition.String())
		if verboseFlag {
			fmt.Printf("    Edition ID: %s\n", edition.EditionID)
			fmt.Printf("    Product: %s\n", edition.ProductName)
			fmt.Printf("    Version: %d.%d.%d\n", edition.MajorVersion, edition.MinorVersion, edition.Build)
			fmt.Printf("    Size: %.1f GB\n", float64(edition.TotalBytes)/(1024*1024*1024))
			fmt.Println()
		}
	}

	return nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Validate inputs
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return fmt.Errorf("ISO file not found: %s", isoPath)
	}

	// Use TUI mode by default unless disabled or disk is specified (CLI mode)
	if tuiFlag && diskID == "" {
		return tui.RunTUI(isoPath)
	}

	// CLI mode requires disk to be specified
	if diskID == "" {
		return fmt.Errorf("disk flag is required when using CLI mode (use --tui=false --disk=diskX)")
	}

	dm := disk.NewManager()
	parser := wim.NewParser()
	extractor := wim.NewExtractor()
	bootloaderMgr := bootloader.NewUEFIBootloader()

	// Set up context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nReceived interrupt signal, cancelling...")
		cancel()
	}()

	fmt.Println("ember2go - Creating portable Windows installation")
	fmt.Println(strings.Repeat("=", 60))

	// Check dependencies first
	if err := checkDependencies(dm, parser, bootloaderMgr); err != nil {
		return err
	}

	// Validate disk
	if !dm.IsDiskIdentifierValid(diskID) {
		return fmt.Errorf("invalid disk identifier: %s", diskID)
	}

	// Mount and analyze ISO
	fmt.Printf("Mounting ISO: %s\n", isoPath)
	var mountPath string
	var wimInfo *wim.WIMInfo
	var selectedEdition *wim.WindowsEdition

	// Ensure cleanup happens even on early returns
	defer func() {
		if mountPath != "" {
			dm.UnmountISO(mountPath)
		}
	}()

	var err error
	mountPath, err = dm.MountISO(isoPath)
	if err != nil {
		return fmt.Errorf("failed to mount ISO: %w", err)
	}

	fmt.Println("Analyzing Windows ISO...")
	wimInfo, err = parser.GetWIMInfo(mountPath)
	if err != nil {
		return fmt.Errorf("failed to read Windows editions from ISO: %w", err)
	}

	// Show available editions
	fmt.Println("\nAvailable Windows editions:")
	for _, edition := range wimInfo.Images {
		fmt.Printf("  %s\n", edition.String())
	}

	// Validate edition index
	if editionIdx < 1 || editionIdx > len(wimInfo.Images) {
		return fmt.Errorf("invalid edition index: %d (available: 1-%d)", editionIdx, len(wimInfo.Images))
	}

	selectedEdition = wimInfo.Images[editionIdx-1]
	fmt.Printf("\nSelected: %s\n", selectedEdition.DisplayName)

	// Format disk section
	fmt.Printf("\nFormatting disk: %s\n", diskID)
	fmt.Println("⚠️  This will ERASE ALL DATA on the selected disk!")

	// Confirm operation
	if !forceFlag {
		fmt.Print("Continue? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Format disk
	bootPartition, windowsPartition, err := dm.FormatDisk(diskID)
	if err != nil {
		return fmt.Errorf("failed to format disk: %w", err)
	}

	fmt.Printf("Created partitions: %s, %s\n", bootPartition, windowsPartition)

	// Mount partitions with proper cleanup
	fmt.Println("Mounting partitions...")
	bootMount, err := dm.MountPartition(bootPartition)
	if err != nil {
		return fmt.Errorf("failed to mount boot partition: %w", err)
	}

	windowsMount, err := dm.MountPartition(windowsPartition)
	if err != nil {
		dm.UnmountPartition(bootPartition) // Cleanup boot partition
		return fmt.Errorf("failed to mount windows partition: %w", err)
	}

	// Ensure partitions are unmounted even on early returns
	defer func() {
		dm.UnmountPartition(bootPartition)
		dm.UnmountPartition(windowsPartition)
	}()

	if bootMount == "" || windowsMount == "" {
		return fmt.Errorf("failed to mount partitions")
	}

	// Extract Windows files
	fmt.Println("Extracting Windows files (this may take 20-30 minutes)...")
	options := &wim.ExtractionOptions{
		NoACLs:              true,
		NoAttributes:        true,
		IncludeInvalidNames: true,
		ProgressCallback:    extractor.GetExtractionProgress(),
	}

	err = extractor.ExtractWindowsEdition(ctx, wimInfo, selectedEdition, windowsMount, options)
	if err != nil {
		return fmt.Errorf("failed to extract Windows files: %w", err)
	}

	fmt.Println("\nWindows extraction completed!")

	// Create bootloader
	fmt.Println("Creating bootloader...")
	diskUUID, _ := dm.GetDiskUUID(diskID)
	if diskUUID == "" {
		diskUUID = "unknown"
	}
	bootUUID, _ := dm.GetPartitionUUID(bootPartition)
	if bootUUID == "" {
		bootUUID = "unknown"
	}
	windowsUUID, _ := dm.GetPartitionUUID(windowsPartition)
	if windowsUUID == "" {
		windowsUUID = "unknown"
	}

	bootloaderSuccess := false
	err = bootloaderMgr.CreateBootloader(bootMount, windowsMount, diskUUID, bootUUID, windowsUUID)
	if err != nil {
		fmt.Printf("Warning: bootloader creation had issues: %v\n", err)
		bootloaderSuccess = false
	} else {
		bootloaderSuccess = true
	}

	if bootloaderSuccess {
		// Success path
		fmt.Println("\n✅ Windows To Go creation completed!")
		fmt.Printf("Boot partition: %s\n", bootMount)
		fmt.Printf("Windows partition: %s\n", windowsMount)
		fmt.Println("\nTo use:")
		fmt.Println("1. Safely eject the USB drive")
		fmt.Println("2. Boot from USB in UEFI mode")
		fmt.Println("3. Windows should start automatically")
		return nil
	} else {
		// Partial success path
		fmt.Println("⚠️  Windows extracted but bootloader creation had issues")
		fmt.Println("You may need to create the bootloader manually")
		return nil
	}
}

func runInstallDeps(cmd *cobra.Command, args []string) error {
	fmt.Println("Installing ember2go dependencies...")

	// Install wimlib
	fmt.Println("Installing wimlib...")
	if err := installWimlib(); err != nil {
		return fmt.Errorf("failed to install wimlib: %w", err)
	}

	// Install hivex for advanced bootloader features
	fmt.Println("Installing hivex...")
	if err := installHivex(); err != nil {
		fmt.Printf("Warning: failed to install hivex: %v\n", err)
		fmt.Println("Basic bootloader functionality will still work.")
	}

	// Install any other bootloader dependencies
	bootloaderMgr := bootloader.NewUEFIBootloader()
	fmt.Println("Installing bootloader dependencies...")
	if err := bootloaderMgr.InstallBootloaderDependencies(); err != nil {
		fmt.Printf("Warning: failed to install some bootloader dependencies: %v\n", err)
	}
	fmt.Println("Basic bootloader functionality available")
	fmt.Println("For advanced features, consider installing:")
	fmt.Println("  brew install hivex")

	fmt.Println("\nDependency installation completed!")
	fmt.Println("You may need to restart your terminal for changes to take effect.")

	return nil
}

func checkDependencies(dm *disk.Manager, parser *wim.Parser, bootloaderMgr *bootloader.UEFIBootloader) error {
	fmt.Println("Checking dependencies...")

	if err := dm.CheckDiskutilAvailable(); err != nil {
		return fmt.Errorf("diskutil not available: %w", err)
	}

	if err := dm.CheckHdiutilAvailable(); err != nil {
		return fmt.Errorf("hdiutil not available: %w", err)
	}

	// Check wimlib
	if err := parser.CheckWIMLibInstalled(); err != nil {
		if !forceFlag {
			fmt.Print("wimlib not found. Install it now? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				return fmt.Errorf("wimlib is required for Windows image extraction")
			}
		}
		fmt.Println("Installing wimlib...")
		if err := installWimlib(); err != nil {
			return fmt.Errorf("failed to install wimlib: %w", err)
		}
		// Check again after installation
		if err := parser.CheckWIMLibInstalled(); err != nil {
			return fmt.Errorf("wimlib installation failed: %w", err)
		}
	}

	fmt.Println("✅ wimlib found")

	// Check BCD tools (hivex) for better bootloader creation
	if err := bootloaderMgr.CheckBootloaderTools(); err != nil {
		if !forceFlag {
			fmt.Print("⚠️  hivex not found. This improves bootloader creation. Install it now? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
				fmt.Println("Installing hivex...")
				if err := installHivex(); err != nil {
					fmt.Printf("Warning: failed to install hivex: %v\n", err)
					fmt.Println("Bootloader creation may be limited.")
				} else {
					fmt.Println("✅ hivex installed")
				}
			} else {
				fmt.Println("⚠️  BCD tools not found. Bootloader creation may be limited.")
			}
		} else {
			fmt.Println("⚠️  BCD tools not found. Bootloader creation may be limited.")
			fmt.Println("Consider installing: brew install hivex")
		}
	} else {
		fmt.Println("✅ hivex found")
	}

	return nil
}

func installWimlib() error {
	cmd := exec.Command("brew", "install", "wimlib")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install wimlib via Homebrew: %w", err)
	}
	return nil
}

func installHivex() error {
	cmd := exec.Command("brew", "install", "hivex")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install hivex via Homebrew: %w", err)
	}
	return nil
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("ember2go version %s\n", version)
	fmt.Printf("Built: %s\n", buildTime)
	fmt.Printf("Commit: %s\n", gitCommit)
	fmt.Printf("Built by: %s\n", builtBy)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Show ARM64 optimization status
	if runtime.GOARCH == "arm64" {
		fmt.Printf("ARM64 optimizations: enabled (LSE, Crypto)\n")
	}

	fmt.Printf("CPU cores: %d\n", runtime.NumCPU())
	fmt.Printf("Compiler: %s\n", runtime.Compiler)
}

