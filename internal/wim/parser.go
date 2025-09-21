package wim

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/entro314-labs/ember2go/internal/performance"
)

// Parser handles WIM file parsing operations
type Parser struct{}

// NewParser creates a new WIM parser
func NewParser() *Parser {
	return &Parser{}
}

// CheckWIMLibInstalled checks if wimlib tools are installed
func (p *Parser) CheckWIMLibInstalled() error {
	cmd := exec.Command("wiminfo", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wimlib not installed. Run: brew install wimlib")
	}
	return nil
}

// GetWIMInfo extracts information about a WIM file from an ISO
func (p *Parser) GetWIMInfo(isoMountPath string) (*WIMInfo, error) {
	wimPath := filepath.Join(isoMountPath, "sources", "install.wim")

	if _, err := os.Stat(wimPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("install.wim not found at %s", wimPath)
	}

	return p.parseWIMFile(wimPath)
}

// parseWIMFile parses a WIM file and extracts Windows editions
func (p *Parser) parseWIMFile(wimPath string) (*WIMInfo, error) {
	cmd := exec.Command("wiminfo", wimPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run wiminfo: %w", err)
	}

	return p.parseWIMInfoOutput(string(output), wimPath)
}

// parseWIMInfoOutput parses the output from wiminfo command (optimized)
func (p *Parser) parseWIMInfoOutput(output, wimPath string) (*WIMInfo, error) {
	// Use buffered reader for better performance
	reader := strings.NewReader(output)
	scanner := bufio.NewScanner(reader)

	// Pre-allocate slices with estimated capacity
	var images []*WindowsEdition
	images = make([]*WindowsEdition, 0, 8) // Typical ISOs have 4-8 editions

	var wimInfo *WIMInfo
	var currentImage *WindowsEdition

	inWIMInfo := false
	inImageSection := false
	skipLine := false

	// Use string builder for efficient parsing
	sb := performance.GetStringBuilder()
	defer performance.PutStringBuilder(sb)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "WIM Information") {
			inWIMInfo = true
			inImageSection = false
			skipLine = true
			wimInfo = &WIMInfo{Path: wimPath}
			continue
		}

		if strings.HasPrefix(line, "Available Images") {
			inWIMInfo = false
			inImageSection = true
			skipLine = true
			continue
		}

		if skipLine {
			skipLine = false
			continue
		}

		// Parse WIM header information
		if inWIMInfo && line != "" && !strings.HasPrefix(line, "-") {
			p.parseWIMHeaderLine(line, wimInfo)
		}

		// Parse image information (optimized)
		if inImageSection {
			if line == "" {
				if currentImage != nil && currentImage.Index > 0 {
					images = append(images, currentImage)
					currentImage = nil
				}
			} else if line != "" && !strings.HasPrefix(line, "-") {
				if currentImage == nil {
					currentImage = &WindowsEdition{}
				}
				p.parseImageLine(line, currentImage)
			}
		}
	}

	// Add the last image if it exists
	if currentImage != nil && currentImage.Index > 0 {
		images = append(images, currentImage)
	}

	if wimInfo == nil {
		return nil, fmt.Errorf("failed to parse WIM information")
	}

	wimInfo.Images = images
	return wimInfo, nil
}

// parseWIMHeaderLine parses a line from the WIM header section
func (p *Parser) parseWIMHeaderLine(line string, wimInfo *WIMInfo) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "GUID":
		wimInfo.GUID = value
	case "Version":
		wimInfo.Version = value
	case "Image Count":
		wimInfo.ImageCount = value
	case "Compression":
		wimInfo.Compression = value
	case "Chunk Size":
		wimInfo.ChunkSize = value
	case "Part Number":
		if parts := strings.Split(value, "/"); len(parts) == 2 {
			if part1, err := strconv.Atoi(parts[0]); err == nil {
				if part2, err := strconv.Atoi(parts[1]); err == nil {
					wimInfo.PartNumber = []int{part1, part2}
				}
			}
		}
	case "Boot Index":
		wimInfo.BootIndex = value
	case "Size":
		if sizeParts := strings.Fields(value); len(sizeParts) > 0 {
			if size, err := strconv.ParseInt(sizeParts[0], 10, 64); err == nil {
				wimInfo.Size = size
			}
		}
	case "Attributes":
		wimInfo.Attributes = value
	}
}

// parseImageLine parses a line from an image section
func (p *Parser) parseImageLine(line string, image *WindowsEdition) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "Index":
		if idx, err := strconv.Atoi(value); err == nil {
			image.Index = idx
		}
	case "Name":
		image.Name = value
	case "Description":
		image.Description = value
	case "Display Name":
		image.DisplayName = value
	case "Display Description":
		image.DisplayDescription = value
	case "Edition ID":
		image.EditionID = value
	case "Architecture":
		image.Architecture = value
	case "Languages":
		image.Languages = value
	case "Default Language":
		image.DefaultLanguage = value
	case "Product Name":
		image.ProductName = value
	case "Installation Type":
		image.InstallationType = value
	case "Product Type":
		image.ProductType = value
	case "Major Version":
		if ver, err := strconv.Atoi(value); err == nil {
			image.MajorVersion = ver
		}
	case "Minor Version":
		if ver, err := strconv.Atoi(value); err == nil {
			image.MinorVersion = ver
		}
	case "Build":
		if build, err := strconv.Atoi(value); err == nil {
			image.Build = build
		}
	case "Service Pack Build":
		if build, err := strconv.Atoi(value); err == nil {
			image.ServicePackBuild = build
		}
	case "Service Pack Level":
		image.ServicePackLevel = value
	case "Directory Count":
		if count, err := strconv.Atoi(value); err == nil {
			image.DirectoryCount = count
		}
	case "File Count":
		if count, err := strconv.Atoi(value); err == nil {
			image.FileCount = count
		}
	case "Total Bytes":
		if bytes, err := strconv.ParseInt(value, 10, 64); err == nil {
			image.TotalBytes = bytes
		}
	case "Hard Link Bytes":
		if bytes, err := strconv.ParseInt(value, 10, 64); err == nil {
			image.HardLinkBytes = bytes
		}
	case "Creation Time":
		image.CreationTime = value
	case "Last Modification Time":
		image.LastModTime = value
	case "System Root":
		image.SystemRoot = value
	case "Flags":
		image.Flags = value
	case "WIMBoot compatible":
		image.WIMBootCompatible = strings.ToLower(value) == "true" || value == "1"
	}
}