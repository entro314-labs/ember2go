package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"

	"github.com/entro314-labs/ember2go/internal/bootloader"
	"github.com/entro314-labs/ember2go/internal/disk"
	"github.com/entro314-labs/ember2go/internal/wim"
)

func LoadDisks() tea.Cmd {
	return func() tea.Msg {
		dm := disk.NewManager()

		if err := dm.CheckDiskutilAvailable(); err != nil {
			return ErrorMsg{Error: fmt.Errorf("diskutil not available: %w", err)}
		}

		disks, err := dm.GetRemovableDisks()
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to get removable disks: %w", err)}
		}

		return DisksLoadedMsg{Disks: disks}
	}
}

func LoadEditions(isoPath string) tea.Cmd {
	return func() tea.Msg {
		dm := disk.NewManager()
		parser := wim.NewParser()

		if err := dm.CheckHdiutilAvailable(); err != nil {
			return ErrorMsg{Error: fmt.Errorf("hdiutil not available: %w", err)}
		}

		if err := parser.CheckWIMLibInstalled(); err != nil {
			return ErrorMsg{Error: fmt.Errorf("wimlib not installed: %w", err)}
		}

		mountPath, err := dm.MountISO(isoPath)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to mount ISO: %w", err)}
		}
		defer dm.UnmountISO(mountPath)

		wimInfo, err := parser.GetWIMInfo(mountPath)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to analyze Windows editions: %w", err)}
		}

		return EditionsLoadedMsg{Editions: wimInfo.Images}
	}
}

func StartCreation(selectedDisk *disk.Disk, selectedEdition *wim.WindowsEdition, isoPath string) tea.Cmd {
	return func() tea.Msg {
		return createWindowsToGo(selectedDisk, selectedEdition, isoPath)
	}
}

func createWindowsToGo(selectedDisk *disk.Disk, selectedEdition *wim.WindowsEdition, isoPath string) tea.Msg {
	dm := disk.NewManager()
	parser := wim.NewParser()
	extractor := wim.NewExtractor()

	// Progress tracking
	updateProgress := func(current, total int, stage string) {
		// This is called within the creation process to update progress
		// The progress will be sent when we return from this function
	}

	// Stage 1: Format disk (10% of progress)
	updateProgress(5, 100, "Formatting disk...")
	bootPartition, windowsPartition, err := dm.FormatDisk(selectedDisk.Identifier)
	if err != nil {
		return ErrorMsg{Error: fmt.Errorf("failed to format disk: %w", err)}
	}

	updateProgress(10, 100, "Mounting partitions...")

	// Stage 2: Mount partitions (15% of progress)
	bootMount, err := dm.MountPartition(bootPartition)
	if err != nil {
		return ErrorMsg{Error: fmt.Errorf("failed to mount boot partition: %w", err)}
	}

	windowsMount, err := dm.MountPartition(windowsPartition)
	if err != nil {
		dm.UnmountPartition(bootPartition) // Cleanup
		return ErrorMsg{Error: fmt.Errorf("failed to mount windows partition: %w", err)}
	}

	// Ensure cleanup on exit
	defer func() {
		dm.UnmountPartition(bootPartition)
		dm.UnmountPartition(windowsPartition)
	}()

	updateProgress(15, 100, "Mounting ISO...")

	// Stage 3: Mount ISO and get WIM info (20% of progress)
	mountPath, err := dm.MountISO(isoPath)
	if err != nil {
		return ErrorMsg{Error: fmt.Errorf("failed to mount ISO: %w", err)}
	}
	defer dm.UnmountISO(mountPath)

	wimInfo, err := parser.GetWIMInfo(mountPath)
	if err != nil {
		return ErrorMsg{Error: fmt.Errorf("failed to analyze Windows image: %w", err)}
	}

	updateProgress(20, 100, "Extracting Windows files...")

	// Stage 4: Extract Windows files (20% -> 85% of progress)
	progressCallback := func(current, total int64, message string) {
		if total > 0 {
			// Map extraction progress to 20-85% of total progress
			extractionPercent := float64(current) / float64(total)
			_ = 20 + int(extractionPercent * 65) // overallPercent for future use
			// Note: We can't send progress updates from within this callback
			// in the current architecture, but the message shows current status
		}
	}

	options := &wim.ExtractionOptions{
		NoACLs:              true,
		NoAttributes:        true,
		IncludeInvalidNames: true,
		ProgressCallback:    progressCallback,
	}

	err = extractor.ExtractWindowsEdition(nil, wimInfo, selectedEdition, windowsMount, options)
	if err != nil {
		return ErrorMsg{Error: fmt.Errorf("failed to extract Windows files: %w", err)}
	}

	updateProgress(85, 100, "Creating bootloader...")

	// Stage 5: Create bootloader (85% -> 95% of progress)
	bootloaderMgr := bootloader.NewUEFIBootloader()

	diskUUID, _ := dm.GetDiskUUID(selectedDisk.Identifier)
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

	err = bootloaderMgr.CreateBootloader(bootMount, windowsMount, diskUUID, bootUUID, windowsUUID)
	if err != nil {
		// Don't fail completely, just warn
		return ProgressUpdateMsg{
			Current: 95,
			Total:   100,
			Stage:   fmt.Sprintf("Warning: bootloader creation had issues: %v", err),
		}
	}

	updateProgress(95, 100, "Finalizing...")

	// Stage 6: Complete (100%)
	updateProgress(100, 100, "Creation completed!")

	// Return completion message
	return CompletionMsg{}
}