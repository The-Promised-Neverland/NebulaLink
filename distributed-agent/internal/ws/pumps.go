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
func (a *Agent) processorPump() {
	defer func() {
		logger.Log.Info("Dispatch pump stopped")
	}()
	for {
		select {
		case msg := <-a.incomingCh:
			if msg.Msg != nil {
				messageRec := *msg.Msg
				if handler, ok := a.Handlers[messageRec.Type]; ok {
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

// handleBinaryChunk writes binary chunks to the temp file
func (a *Agent) handleBinaryChunk(chunk []byte) {
	// Check if we have an active P2P connection receiving
	if a.P2PClient != nil {
		// Check if there's an active P2P connection
		// If P2P is active, we should be receiving over TCP, not WebSocket
		// So this is likely a relay transfer
	}
	
	if a.TempFile == nil {
		logger.Log.Warn("Received binary chunk but no temp file open, dropping chunk", "size", len(chunk))
		return
	}
	written, err := a.TempFile.Write(chunk)
	if err != nil {
		logger.Log.Error("Failed to write chunk to temp file", "err", err, "written", written)
		return
	}
	logger.Log.Debug("Wrote chunk to temp file", "bytes", written, "totalChunkSize", len(chunk))
}

// StartTransfer creates a temp file for receiving transfer data
func (a *Agent) StartTransfer(sourceAgentID string) error {
	if a.TempFile != nil {
		a.TempFile.Close()
		os.Remove(a.tempFilePath)
	}
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "transfer_*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	a.TempFile = tempFile
	a.tempFilePath = tempFile.Name()
	a.sourceAgent = sourceAgentID
	logger.Log.Info("Started transfer", "sourceAgent", sourceAgentID, "tempFile", a.tempFilePath)
	return nil
}

// CompleteTransfer closes the temp file, extracts the tar, and cleans up
func (a *Agent) CompleteTransfer() error {
	if a.TempFile == nil {
		return fmt.Errorf("no active transfer to complete")
	}
	if err := a.TempFile.Close(); err != nil {
		logger.Log.Error("Failed to close temp file", "err", err)
	}
	tempPath := a.tempFilePath
	sourceAgent := a.sourceAgent
	a.TempFile = nil
	a.tempFilePath = ""
	a.sourceAgent = ""
	if err := a.extractTarToSharedFolder(tempPath, sourceAgent); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to extract tar: %w", err)
	}
	if err := os.Remove(tempPath); err != nil {
		logger.Log.Warn("Failed to remove temp file", "path", tempPath, "err", err)
	}
	logger.Log.Info("Completed transfer", "sourceAgent", sourceAgent)
	return nil
}

// extractTarToSharedFolder extracts a tar file to the shared/transfers folder
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
	go a.readPump()
	go a.writePump()
	go a.processorPump()
	logger.Log.Info("All pumps started")
}
