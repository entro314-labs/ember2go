package wim

import "fmt"

// WindowsEdition represents a Windows edition in a WIM file
type WindowsEdition struct {
	Index              int    `json:"index"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	DisplayName        string `json:"display_name"`
	DisplayDescription string `json:"display_description"`
	EditionID          string `json:"edition_id"`
	Architecture       string `json:"architecture"`
	Languages          string `json:"languages"`
	DefaultLanguage    string `json:"default_language"`
	ProductName        string `json:"product_name"`
	InstallationType   string `json:"installation_type"`
	ProductType        string `json:"product_type"`
	MajorVersion       int    `json:"major_version"`
	MinorVersion       int    `json:"minor_version"`
	Build              int    `json:"build"`
	ServicePackBuild   int    `json:"service_pack_build"`
	ServicePackLevel   string `json:"service_pack_level"`
	DirectoryCount     int    `json:"directory_count"`
	FileCount          int    `json:"file_count"`
	TotalBytes         int64  `json:"total_bytes"`
	HardLinkBytes      int64  `json:"hard_link_bytes"`
	CreationTime       string `json:"creation_time"`
	LastModTime        string `json:"last_modification_time"`
	SystemRoot         string `json:"system_root"`
	Flags              string `json:"flags"`
	WIMBootCompatible  bool   `json:"wimboot_compatible"`
}

// String returns a human-readable representation of the Windows edition
func (w *WindowsEdition) String() string {
	return fmt.Sprintf("%d: %s (%s)", w.Index, w.DisplayName, w.Architecture)
}

// WIMInfo represents information about a WIM file
type WIMInfo struct {
	Path        string            `json:"path"`
	GUID        string            `json:"guid"`
	Version     string            `json:"version"`
	ImageCount  string            `json:"image_count"`
	Compression string            `json:"compression"`
	ChunkSize   string            `json:"chunk_size"`
	PartNumber  []int             `json:"part_number"`
	Size        int64             `json:"size"`
	BootIndex   string            `json:"boot_index"`
	Attributes  string            `json:"attributes"`
	Images      []*WindowsEdition `json:"images"`
}

// GetEditionByIndex returns a Windows edition by its index
func (w *WIMInfo) GetEditionByIndex(index int) *WindowsEdition {
	for _, edition := range w.Images {
		if edition.Index == index {
			return edition
		}
	}
	return nil
}

// ProgressCallback is called during WIM extraction to report progress
type ProgressCallback func(current, total int64, message string)

// ExtractionOptions holds options for WIM extraction
type ExtractionOptions struct {
	NoACLs              bool
	NoAttributes        bool
	IncludeInvalidNames bool
	ProgressCallback    ProgressCallback
}
