package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"

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
		// TODO: Implement actual creation process
		// For now, just return initial progress
		return ProgressUpdateMsg{
			Current: 0,
			Total:   100,
			Stage:   "Starting creation process...",
		}
	}
}