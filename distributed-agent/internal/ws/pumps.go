package ws

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/gorilla/websocket"
)

const (
	readDeadline  = 70 * time.Second
	writeDeadline = 10 * time.Second
	pongWait      = 60 * time.Second
)

// setupPingPongHandlers sets up ping/pong handlers to maintain connection health
func (a *Agent) setupPingPongHandlers() {
	if a.Conn == nil {
		return
	}
	a.Conn.SetReadDeadline(time.Now().Add(pongWait))
	a.Conn.SetPingHandler(func(appData string) error {
		if a.Conn != nil {
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
			err := a.Conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeDeadline))
			if err != nil {
				logger.Log.Error("Failed to send pong", "err", err)
				return err
			}
		}
		return nil
	})
	a.Conn.SetPongHandler(func(appData string) error {
		if a.Conn != nil {
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
		}
		return nil
	})
}

// readPump handles incoming messages from master
func (a *Agent) readPump() {
	defer func() {
		logger.Log.Info("Read pump stopped")
	}()
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if a.Conn == nil {
				return
			}
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
			msgType, msgBytes, err := a.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Log.Error("WebSocket read error", "err", err)
				}
				a.Close()
				return
			}
			switch msgType {
			case websocket.TextMessage:
				var msg models.Message
				if err := json.Unmarshal(msgBytes, &msg); err != nil {
					logger.Log.Warn("Unmarshalling error: TEXT", "err", err)
					continue
				}
				select {
				case a.incomingCh <- Outbound{Msg: &msg}:
				case <-a.ctx.Done():
					return
				default:
					// TODO: Handling backpressure
					logger.Log.Warn("Incoming channel full for %s, dropping message\n")
				}
			case websocket.BinaryMessage:
				select {
				case a.incomingCh <- Outbound{Binary: msgBytes}:
				case <-a.ctx.Done():
					return
				default:
					// TODO: Handling backpressure
					logger.Log.Warn("Incoming channel full for %s, dropping message\n")
				}
			default:
				logger.Log.Warn("Recieved mssg dropped. Neither TEXT/BINARY")
			}
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}
}

// writePump handles outgoing messages to master
func (a *Agent) writePump() {
	defer func() {
		logger.Log.Info("Write pump stopped")
	}()
	for {
		select {
		case msg := <-a.sendCh:
			if a.Conn == nil {
				return
			}
			a.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if msg.Msg != nil {
				bytes, err := json.Marshal(*msg.Msg)
				if err != nil {
					logger.Log.Error("Marshalling error", "err", err)
					continue
				}
				if err := a.Conn.WriteMessage(websocket.TextMessage, bytes); err != nil {
					logger.Log.Error("Write error: TEXT", "err", err)
					a.Close()
					return
				}
			} else {
				if err := a.Conn.WriteMessage(websocket.BinaryMessage, msg.Binary); err != nil {
					logger.Log.Error("Write error: BINARY", "err", err)
					a.Close()
					return
				}
			}
		case <-a.ctx.Done():
			if a.Conn != nil {
				a.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
			}
			return
		}
	}
}

// dispatchPump dispatches incoming messages to registered handlers
func (a *Agent) dispatchPump() {
	defer func() {
		logger.Log.Info("Dispatch pump stopped")
	}()
	for {
		select {
		case msg := <-a.incomingCh:
			if msg.Msg != nil {
				messageRec := *msg.Msg
				if messageRec.Type == models.MasterMsgRelayManager {
					payloadMap, ok := messageRec.Payload.(map[string]interface{})
					if ok {
						if status, hasStatus := payloadMap["status"].(string); hasStatus && status != "" {
							if err := a.handleTransferStatus(&messageRec.Payload); err != nil {
								logger.Log.Warn("Failed to handle transfer status", "err", err)
							}
							continue
						}
					}
				}
				if handler, ok := a.Handlers[messageRec.Type]; ok {
					logger.Log.Debug("Found handler for message type", "type", messageRec.Type)
					if err := handler(&messageRec.Payload); err != nil {
						logger.Log.Error("Handler error", "type", messageRec.Type, "err", err)
					}
				} else {
					logger.Log.Warn("No handler for message type", "type", messageRec.Type, "payload", messageRec.Payload)
				}
			} else {
				a.handleBinaryChunk(msg.Binary)
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// handleTransferStatus handles filesystem transfer status messages
func (a *Agent) handleTransferStatus(payload *interface{}) error {
	var status, agentID string
	if payloadMap, ok := (*payload).(map[string]interface{}); ok {
		status, _ = payloadMap["status"].(string)
		agentID, _ = payloadMap["agent_id"].(string)
	}
	if status == "" {
		logger.Log.Error("status not found for streaming...")
		return nil
	}
	a.processTransferStatus(status, agentID)
	return nil
}

// processTransferStatus processes the transfer status and manages transfer state
func (a *Agent) processTransferStatus(status, sourceAgentID string) {
	a.transferMutex.Lock()
	defer a.transferMutex.Unlock()
	switch status {
	case "initiated":
		// TODO: Later we will tag the chunk with the transferID to allow multiple streams sends
		if a.transferState != nil && a.transferState.isActive {
			logger.Log.Warn("Transfer already in progress, ignoring new initiated message", "sourceAgent", sourceAgentID)
			return
		}	
		tempDir := os.TempDir()
		tempFile := filepath.Join(tempDir, fmt.Sprintf("nebulalink-transfer-%s-%d.tar", sourceAgentID, time.Now().Unix()))
		file, err := os.Create(tempFile)
		if err != nil {
			logger.Log.Error("Failed to create temp file for transfer", "err", err, "path", tempFile)
			return
		}
		a.transferState = &transferState{
			file:        file,
			isActive:    true,
			sourceAgent: sourceAgentID,
			tempPath:    tempFile,
		}
		logger.Log.Info("Transfer initiated", "sourceAgent", sourceAgentID, "tempFile", tempFile)
	case "running":
		if a.transferState != nil && a.transferState.isActive {
			logger.Log.Info("Transfer in progress", "sourceAgent", sourceAgentID)
		}
	case "completed":
		if a.transferState == nil || !a.transferState.isActive {
			logger.Log.Warn("Received completed status but no active transfer", "sourceAgent", sourceAgentID)
			return
		}
		if err := a.transferState.file.Close(); err != nil {
			logger.Log.Error("Failed to close temp file", "err", err)
		}
		if err := a.extractTarToSharedFolder(a.transferState.tempPath, sourceAgentID); err != nil {
			logger.Log.Error("Failed to extract tar", "err", err, "tempFile", a.transferState.tempPath)
		} else {
			logger.Log.Info("Transfer completed and extracted", "sourceAgent", sourceAgentID)
		}
		if err := os.Remove(a.transferState.tempPath); err != nil {
			logger.Log.Warn("Failed to remove temp file", "err", err, "path", a.transferState.tempPath)
		}
		a.transferState = nil
	}
}

// handleBinaryChunk writes binary chunks to the active transfer file
func (a *Agent) handleBinaryChunk(chunk []byte) {
	a.transferMutex.Lock()
	defer a.transferMutex.Unlock()
	if a.transferState == nil || !a.transferState.isActive {
		logger.Log.Warn("Received binary chunk but no active transfer, dropping chunk", "size", len(chunk))
		return
	}
	if a.transferState.file == nil {
		logger.Log.Error("Transfer file is nil, cannot write chunk")
		return
	}
	written, err := a.transferState.file.Write(chunk)
	if err != nil {
		logger.Log.Error("Failed to write chunk to temp file", "err", err, "written", written)
		return
	}
	logger.Log.Debug("Wrote chunk to temp file", "bytes", written, "totalChunkSize", len(chunk))
}

// extractTarToSharedFolder extracts a tar file to the shared folder
func (a *Agent) extractTarToSharedFolder(tarPath, sourceAgentID string) error {
	sharedPath, err := a.Config.SharedFolderPath()
	if err != nil {
		return fmt.Errorf("failed to get shared folder path: %w", err)
	}
	extractPath := filepath.Join(sharedPath, "transfers", sourceAgentID)
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}
	tarFile, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer tarFile.Close()
	tarReader := tar.NewReader(tarFile)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		// Skip empty or root directory entries
		cleanName := filepath.Clean(header.Name)
		if cleanName == "" || cleanName == "." || cleanName == "/" {
			logger.Log.Error("Skipping root/empty tar entry", "name", header.Name)
			continue
		}
		targetPath := filepath.Join(extractPath, cleanName)
		extractPathClean := filepath.Clean(extractPath)
		if !filepath.HasPrefix(targetPath, extractPathClean+string(os.PathSeparator)) && targetPath != extractPathClean {
			logger.Log.Warn("Skipping file with invalid path (outside extract directory)", "path", header.Name, "targetPath", targetPath)
			continue
		}
		if targetPath == extractPathClean {
			logger.Log.Debug("Skipping entry that would overwrite extract directory", "name", header.Name)
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			logger.Log.Info("Extracted directory", "path", targetPath)
		case tar.TypeReg:
			if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
				logger.Log.Warn("Skipping file entry - target path is a directory", "path", targetPath, "headerName", header.Name)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
			logger.Log.Debug("Extracted file", "path", targetPath, "size", header.Size)
		default:
			logger.Log.Warn("Unsupported tar entry type", "type", header.Typeflag, "name", header.Name)
		}
	}
	logger.Log.Info("Successfully extracted tar to shared folder", "sourceAgent", sourceAgentID, "extractPath", extractPath)
	return nil
}

// RunPumps starts all pumps and sets up ping/pong handlers
func (a *Agent) RunPumps() {
	if a.Conn == nil {
		logger.Log.Error("Cannot start pumps: connection is nil")
		return
	}
	a.setupPingPongHandlers()
	go a.readPump()     // Reads messages
	go a.writePump()    // Writes messages
	go a.dispatchPump() // Dispatches to handlers
	logger.Log.Info("All pumps started")
}
