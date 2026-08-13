package disk

import "fmt"

// Disk represents a storage device
type Disk struct {
	Identifier   string `json:"identifier"`
	Size         int64  `json:"size"`
	DeviceName   string `json:"device_name"`
	Removable    bool   `json:"removable"`
	MediaName    string `json:"media_name"`
	DiskUUID     string `json:"disk_uuid"`
	PartitionMap string `json:"partition_map"`
}

// String returns a human-readable representation of the disk
func (d *Disk) String() string {
	sizeGB := d.Size / (1024 * 1024 * 1024)
	return fmt.Sprintf("%s - %s (%dGB)", d.Identifier, d.DeviceName, sizeGB)
}

// Partition represents a disk partition
type Partition struct {
	Identifier string `json:"identifier"`
	Size       int64  `json:"size"`
	MountPoint string `json:"mount_point"`
	FileSystem string `json:"file_system"`
	Label      string `json:"label"`
	UUID       string `json:"uuid"`
}

// DiskutilListOutput represents the output from `diskutil list -plist`
type DiskutilListOutput struct {
	AllDisksAndPartitions []DiskInfo `plist:"AllDisksAndPartitions"`
}

// DiskInfo represents disk information from diskutil
type DiskInfo struct {
	DeviceIdentifier string          `plist:"DeviceIdentifier"`
	Partitions       []PartitionInfo `plist:"Partitions"`
	Size             int64           `plist:"Size"`
}

// PartitionInfo represents partition information from diskutil
type PartitionInfo struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	Size             int64  `plist:"Size"`
	MountPoint       string `plist:"MountPoint"`
}

// DiskutilInfoOutput represents the output from `diskutil info -plist`
type DiskutilInfoOutput struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	MediaName        string `plist:"MediaName"`
	TotalSize        int64  `plist:"TotalSize"`
	Removable        bool   `plist:"Removable"`
	VolumeUUID       string `plist:"VolumeUUID"`
	DiskUUID         string `plist:"DiskUUID"`
	PartitionMapType string `plist:"PartitionMapType"`
	MountPoint       string `plist:"MountPoint"`
}
