package tui

import (
	"github.com/entro314-labs/ember2go/internal/disk"
	"github.com/entro314-labs/ember2go/internal/wim"
)

type DisksLoadedMsg struct {
	Disks []*disk.Disk
}

type EditionsLoadedMsg struct {
	Editions []*wim.WindowsEdition
}

type ProgressUpdateMsg struct {
	Current int
	Total   int
	Stage   string
}

type CompletionMsg struct{}

type ErrorMsg struct {
	Error error
}