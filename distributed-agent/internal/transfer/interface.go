package transfer

import (
	"os"
)

type TransferMode string
const (
	ModeP2P   TransferMode = "p2p"
	ModeRelay TransferMode = "relay"
)

type TransferContext struct {
	SourceAgentID    string
	RequestingAgentID string
	TempFile         *os.File
	TempFilePath     string
	Mode             TransferMode
	ConnectionID     string
	ChunkCount       int
	TotalBytes       int64
}

type Transferer interface {
	Send(path string, requestingAgentID string) error
	Receive(sourceAgentID string) error
	WriteChunk(chunk []byte) error
	Complete() error
	GetMode() TransferMode
}


type Extractor interface {
	ExtractTar(tarPath string, sourceAgentID string) error
}


