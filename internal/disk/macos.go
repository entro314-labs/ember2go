package disk

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"howett.net/plist"
)

// Manager handles disk operations
type Manager struct{}

// NewManager creates a new disk manager
func NewManager() *Manager {
	return &Manager{}
}

// GetRemovableDisks returns a list of removable disks (optimized for ARM64)
func (m *Manager) GetRemovableDisks() ([]*Disk, error) {
	// Get list of all disks
	cmd := exec.Command("diskutil", "list", "-plist")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list disks: %w", err)
	}

	var listOutput DiskutilListOutput
	if _, err := plist.Unmarshal(output, &listOutput); err != nil {
		return nil, fmt.Errorf("failed to parse diskutil list output: %w", err)
	}

	// Pre-allocate slice with estimated capacity
	removableDisks := make([]*Disk, 0, len(listOutput.AllDisksAndPartitions))

	// Use goroutines for concurrent disk info gathering on multi-core ARM systems
	type diskResult struct {
		disk *Disk
		err  error
	}

	resultChan := make(chan diskResult, len(listOutput.AllDisksAndPartitions))
	var wg sync.WaitGroup

	// Limit concurrency based on CPU cores (ARM Macs benefit from parallelization)
	maxWorkers := runtime.NumCPU()
	if maxWorkers > len(listOutput.AllDisksAndPartitions) {
		maxWorkers = len(listOutput.AllDisksAndPartitions)
	}

	semaphore := make(chan struct{}, maxWorkers)

	for _, diskInfo := range listOutput.AllDisksAndPartitions {
		wg.Add(1)
		go func(identifier string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			disk, err := m.getDiskInfo(identifier)
			resultChan <- diskResult{disk: disk, err: err}
		}(diskInfo.DeviceIdentifier)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.err != nil {
			continue // Skip disks we can't get info for
		}

		// Only include removable disks with reasonable size (> 1GB)
		if result.disk.Removable && result.disk.Size > 1024*1024*1024 {
			removableDisks = append(removableDisks, result.disk)
		}
	}

	return removableDisks, nil
}

// getDiskInfo gets detailed information about a specific disk
func (m *Manager) getDiskInfo(identifier string) (*Disk, error) {
	cmd := exec.Command("diskutil", "info", "-plist", identifier)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk info for %s: %w", identifier, err)
	}

	var info DiskutilInfoOutput
	if _, err := plist.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse diskutil info output: %w", err)
	}

	return &Disk{
		Identifier:   info.DeviceIdentifier,
		Size:         info.TotalSize,
		DeviceName:   info.MediaName,
		Removable:    info.Removable,
		MediaName:    info.MediaName,
		DiskUUID:     info.DiskUUID,
		PartitionMap: info.PartitionMapType,
	}, nil
}

// FormatDisk formats a disk with GPT and creates BOOT (FAT32) and WINDOWS (ExFAT) partitions
func (m *Manager) FormatDisk(diskIdentifier string) (bootPartition, windowsPartition string, err error) {
	devicePath := "/dev/" + diskIdentifier
	bootSize := "512m"

	// Unmount all partitions first
	if err := m.unmountDisk(diskIdentifier); err != nil {
		return "", "", fmt.Errorf("failed to unmount disk: %w", err)
	}

	// Partition the disk
	cmd := exec.Command("diskutil", "partitionDisk", devicePath,
		"GPT",
		"FAT32", "BOOT", bootSize,
		"ExFAT", "WINDOWS", "R") // R = remainder of disk

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("failed to partition disk: %w", err)
	}

	// Return partition identifiers (usually diskXs1 and diskXs2)
	bootPartition = diskIdentifier + "s1"
	windowsPartition = diskIdentifier + "s2"

	return bootPartition, windowsPartition, nil
}

// MountISO mounts an ISO file and returns the mount point
func (m *Manager) MountISO(isoPath string) (string, error) {
	cmd := exec.Command("hdiutil", "attach", "-plist", isoPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to mount ISO: %w", err)
	}

	// Parse plist output to get mount point
	var data map[string]interface{}
	if _, err := plist.Unmarshal(output, &data); err != nil {
		return "", fmt.Errorf("failed to parse hdiutil output: %w", err)
	}

	// Extract mount point from system-entities
	if entities, ok := data["system-entities"].([]interface{}); ok {
		for _, entity := range entities {
			if entityMap, ok := entity.(map[string]interface{}); ok {
				if mountPoint, ok := entityMap["mount-point"].(string); ok && mountPoint != "" {
					return mountPoint, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find mount point in hdiutil output")
}

// UnmountISO unmounts an ISO file
func (m *Manager) UnmountISO(mountPoint string) error {
	cmd := exec.Command("hdiutil", "detach", mountPoint)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unmount ISO: %w", err)
	}
	return nil
}

// MountPartition mounts a partition and returns the mount point
func (m *Manager) MountPartition(partitionIdentifier string) (string, error) {
	devicePath := "/dev/" + partitionIdentifier
	cmd := exec.Command("diskutil", "mount", devicePath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to mount partition: %w", err)
	}

	// Get mount point
	return m.getPartitionMountPoint(partitionIdentifier)
}

// UnmountPartition unmounts a partition
func (m *Manager) UnmountPartition(partitionIdentifier string) error {
	devicePath := "/dev/" + partitionIdentifier
	cmd := exec.Command("diskutil", "unmount", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unmount partition: %w", err)
	}
	return nil
}

// GetPartitionUUID gets the UUID of a partition
func (m *Manager) GetPartitionUUID(partitionIdentifier string) (string, error) {
	cmd := exec.Command("diskutil", "info", "-plist", partitionIdentifier)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get partition info: %w", err)
	}

	var info DiskutilInfoOutput
	if _, err := plist.Unmarshal(output, &info); err != nil {
		return "", fmt.Errorf("failed to parse partition info: %w", err)
	}

	return info.VolumeUUID, nil
}

// GetDiskUUID gets the UUID of a disk
func (m *Manager) GetDiskUUID(diskIdentifier string) (string, error) {
	cmd := exec.Command("diskutil", "info", "-plist", diskIdentifier)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get disk info: %w", err)
	}

	var info DiskutilInfoOutput
	if _, err := plist.Unmarshal(output, &info); err != nil {
		return "", fmt.Errorf("failed to parse disk info: %w", err)
	}

	return info.DiskUUID, nil
}

// unmountDisk unmounts all partitions on a disk
func (m *Manager) unmountDisk(diskIdentifier string) error {
	devicePath := "/dev/" + diskIdentifier
	cmd := exec.Command("diskutil", "unmountDisk", devicePath)
	return cmd.Run() // Ignore errors as disk might not be mounted
}

// getPartitionMountPoint gets the mount point of a partition
func (m *Manager) getPartitionMountPoint(partitionIdentifier string) (string, error) {
	cmd := exec.Command("diskutil", "info", "-plist", partitionIdentifier)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get partition info: %w", err)
	}

	var info DiskutilInfoOutput
	if _, err := plist.Unmarshal(output, &info); err != nil {
		return "", fmt.Errorf("failed to parse partition info: %w", err)
	}

	if info.MountPoint == "" {
		return "", fmt.Errorf("partition %s is not mounted", partitionIdentifier)
	}

	return info.MountPoint, nil
}

// CheckDiskutilAvailable checks if diskutil is available
func (m *Manager) CheckDiskutilAvailable() error {
	cmd := exec.Command("diskutil")
	if err := cmd.Run(); err != nil {
		// diskutil without args returns exit code 1 but still works
		// Just check if the command exists
		if cmd.ProcessState != nil {
			return nil // Command was found and executed
		}
		return fmt.Errorf("diskutil not available: %w", err)
	}
	return nil
}

// CheckHdiutilAvailable checks if hdiutil is available
func (m *Manager) CheckHdiutilAvailable() error {
	cmd := exec.Command("hdiutil")
	if err := cmd.Run(); err != nil {
		// hdiutil without args returns exit code 1 but still works
		if cmd.ProcessState != nil {
			return nil // Command was found and executed
		}
		return fmt.Errorf("hdiutil not available: %w", err)
	}
	return nil
}

// IsDiskIdentifierValid checks if a disk identifier is valid
func (m *Manager) IsDiskIdentifierValid(identifier string) bool {
	// Basic validation: should start with "disk" followed by a number
	if !strings.HasPrefix(identifier, "disk") {
		return false
	}

	// Try to get info for this disk
	_, err := m.getDiskInfo(identifier)
	return err == nil
}
