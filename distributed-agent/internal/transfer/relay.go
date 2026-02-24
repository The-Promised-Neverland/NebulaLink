package transfer

import (
	"fmt"
	"os"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

type RelayTransfer struct {
	ctx             *TransferContext
	config          *config.Config
	businessService *service.Service
	agent           *ws.Agent
	extractor       Extractor
}

func NewRelayTransfer(ctx *TransferContext, cfg *config.Config, businessService *service.Service, agent *ws.Agent, extractor Extractor) *RelayTransfer {
	return &RelayTransfer{
		ctx:             ctx,
		config:          cfg,
		businessService: businessService,
		agent:           agent,
		extractor:       extractor,
	}
}

func (r *RelayTransfer) GetMode() TransferMode {
	return ModeRelay
}

func (r *RelayTransfer) Send(path string, requestingAgentID string) error {
	logger.Log.Info("[RELAY] Starting relay mode transfer", "path", path, "target", requestingAgentID)
	dataCh, errCh := r.businessService.StreamRequestedFileSystem(path)
	starterMsg := models.Message{
		Type: models.MasterMsgTransferStatus,
		Payload: map[string]interface{}{
			"status":   "initiated",
			"agent_id": r.config.AgentID(),
		},
	}
	r.agent.Send(ws.Outbound{Msg: &starterMsg})
	logger.Log.Info("[RELAY] Sent 'initiated' status to master, starting to send binary chunks via relay")
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan struct{})
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				statusMsg := models.Message{
					Type: models.MasterMsgTransferStatus,
					Payload: map[string]interface{}{
						"status":   "running",
						"agent_id": r.config.AgentID(),
					},
				}
				r.agent.Send(ws.Outbound{Msg: &statusMsg})
			}
		}
	}()
	totalBytes := 0
	chunkCount := 0
	for chunk := range dataCh {
		totalBytes += len(chunk)
		chunkCount++
		if chunkCount%100 == 0 || chunkCount == 1 {
			logger.Log.Info("[RELAY] Sending binary chunks via relay", "chunk_number", chunkCount, "chunk_bytes", len(chunk), "total_bytes", totalBytes)
		}
		r.agent.Send(ws.Outbound{Binary: chunk})
	}
	close(done)
	select {
	case err := <-errCh:
		if err != nil {
			logger.Log.Error("[RELAY] Stream error occurred", "error", err)
			return err
		}
	default:
	}
	logger.Log.Info("[RELAY] All binary chunks sent via relay", "total_chunks", chunkCount, "total_bytes", totalBytes)
	doneMsg := models.Message{
		Type: models.MasterMsgTransferStatus,
		Payload: map[string]interface{}{
			"status":   "completed",
			"agent_id": r.config.AgentID(),
		},
	}
	r.agent.Send(ws.Outbound{Msg: &doneMsg})
	logger.Log.Info("[RELAY] Sent 'completed' status to master", "total_bytes_sent", totalBytes)
	return nil
}

func (r *RelayTransfer) Receive(sourceAgentID string) error {
	logger.Log.Info("[RELAY] Preparing to receive relay transfer", "sourceAgent", sourceAgentID)
	if err := r.createTempFile(sourceAgentID); err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	logger.Log.Info("[RELAY] Ready to receive binary chunks via relay", "sourceAgent", sourceAgentID)
	return nil
}

func (r *RelayTransfer) WriteChunk(chunk []byte) error {
	if r.ctx.TempFile == nil {
		logger.Log.Warn("Received binary chunk but no temp file open, dropping chunk", "size", len(chunk), "source_agent", r.ctx.SourceAgentID)
		return fmt.Errorf("no temp file open")
	}
	written, err := r.ctx.TempFile.Write(chunk)
	if err != nil {
		logger.Log.Error("[RELAY] Failed to write chunk to temp file", "err", err, "written", written, "chunk_size", len(chunk), "source_agent", r.ctx.SourceAgentID)
		return fmt.Errorf("failed to write chunk: %w", err)
	}
	r.ctx.ChunkCount++
	r.ctx.TotalBytes += int64(written)
	// Log every 100 chunks or first chunk
	if r.ctx.ChunkCount%100 == 0 || r.ctx.ChunkCount == 1 {
		logger.Log.Info("[RELAY] Receiving binary chunks via relay", "chunk_number", r.ctx.ChunkCount, "chunk_bytes", len(chunk), "total_bytes", r.ctx.TotalBytes, "source_agent", r.ctx.SourceAgentID)
	}
	return nil
}

func (r *RelayTransfer) Complete() error {
	return r.completeTransfer()
}

func (r *RelayTransfer) createTempFile(sourceAgentID string) error {
	if r.ctx.TempFile != nil {
		r.ctx.TempFile.Close()
		os.Remove(r.ctx.TempFilePath)
	}
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "transfer_*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	r.ctx.TempFile = tempFile
	r.ctx.TempFilePath = tempFile.Name()
	r.ctx.SourceAgentID = sourceAgentID
	logger.Log.Info("Created temp file for relay transfer", "sourceAgent", sourceAgentID, "tempFile", r.ctx.TempFilePath)
	return nil
}

func (r *RelayTransfer) completeTransfer() error {
	if r.ctx.TempFile == nil {
		return fmt.Errorf("no active transfer to complete")
	}
	if err := r.ctx.TempFile.Close(); err != nil {
		logger.Log.Error("Failed to close temp file", "err", err)
	}
	tempPath := r.ctx.TempFilePath
	sourceAgent := r.ctx.SourceAgentID
	r.ctx.TempFile = nil
	r.ctx.TempFilePath = ""
	r.ctx.SourceAgentID = ""
	if err := r.extractor.ExtractTar(tempPath, sourceAgent); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to extract tar: %w", err)
	}
	if err := os.Remove(tempPath); err != nil {
		logger.Log.Warn("Failed to remove temp file", "path", tempPath, "err", err)
	}
	logger.Log.Info("[RELAY] Relay transfer completed and file extracted", "sourceAgent", sourceAgent, "total_bytes", r.ctx.TotalBytes)
	return nil
}

