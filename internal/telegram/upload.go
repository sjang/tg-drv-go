package telegram

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strconv"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg-drv-go/internal/hash"
	"tg-drv-go/internal/storage"
)

const MaxFileSize = 2 * 1024 * 1024 * 1024 // 2GB

type UploadProgress struct {
	FileID   string  `json:"file_id"`
	FileName string  `json:"file_name"`
	Uploaded int64   `json:"uploaded"`
	Total    int64   `json:"total"`
	Percent  float64 `json:"percent"`
}

type progressCallback struct {
	fileID   string
	fileName string
	onUpdate func(UploadProgress)
}

func (p *progressCallback) Chunk(_ context.Context, state uploader.ProgressState) error {
	progress := UploadProgress{
		FileID:   p.fileID,
		FileName: p.fileName,
		Uploaded: state.Uploaded,
		Total:    state.Total,
	}
	if state.Total > 0 {
		progress.Percent = float64(state.Uploaded) / float64(state.Total) * 100
	}
	if p.onUpdate != nil {
		p.onUpdate(progress)
	}
	return nil
}

func (c *Client) UploadFile(ctx context.Context, folderID, localPath string, onProgress func(UploadProgress)) (*storage.File, error) {
	// Get folder
	folder, err := c.db.GetFolder(folderID)
	if err != nil {
		return nil, fmt.Errorf("get folder: %w", err)
	}

	// Compute hash
	fileHash, fileSize, err := hash.FileHash(localPath)
	if err != nil {
		return nil, fmt.Errorf("hash file: %w", err)
	}

	if fileSize > MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", fileSize, MaxFileSize)
	}

	fileName := filepath.Base(localPath)
	mimeType := mime.TypeByExtension(filepath.Ext(localPath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Check dedup: same file in same folder
	exists, err := c.db.FileExistsInFolder(folderID, fileHash)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("file already exists in this folder (same hash)")
	}

	// Check dedup: file exists in another folder → forward
	dup, err := c.db.FindDuplicate(fileHash)
	if err != nil {
		return nil, err
	}

	if dup != nil {
		return c.forwardFile(ctx, dup, folder, fileName, fileSize, mimeType, fileHash)
	}

	// Upload new file
	upload, err := c.uploader.FromPath(ctx, localPath)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}

	caption := BuildCaption(fileName, mimeType, fileHash, fileSize)

	inputPeer := &tg.InputPeerChannel{
		ChannelID:  folder.ChannelID,
		AccessHash: folder.AccessHash,
	}

	updates, err := c.api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer: inputPeer,
		Media: &tg.InputMediaUploadedDocument{
			File:     upload,
			MimeType: mimeType,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: fileName},
			},
		},
		Message: caption,
	})
	if err != nil {
		return nil, fmt.Errorf("send media: %w", err)
	}

	messageID := extractMessageID(updates)
	if messageID == 0 {
		return nil, fmt.Errorf("could not extract message ID from response")
	}

	file, err := c.db.CreateFile(
		folderID, fileName, fileSize, mimeType, fileHash,
		messageID, "", nil, false, "",
	)
	if err != nil {
		return nil, fmt.Errorf("save file record: %w", err)
	}

	c.logger.Info("uploaded file",
		zap.String("name", fileName),
		zap.Int64("size", fileSize),
		zap.Int("message_id", messageID),
	)
	return file, nil
}

func (c *Client) forwardFile(ctx context.Context, dup *storage.DuplicateInfo, targetFolder *storage.Folder, fileName string, fileSize int64, mimeType, fileHash string) (*storage.File, error) {
	fromPeer := &tg.InputPeerChannel{
		ChannelID:  dup.ChannelID,
		AccessHash: 0, // will need to fetch
	}

	// Get source folder for access hash
	srcFolder, err := c.db.GetFolder(dup.FolderID)
	if err != nil {
		return nil, err
	}
	fromPeer.AccessHash = srcFolder.AccessHash

	toPeer := &tg.InputPeerChannel{
		ChannelID:  targetFolder.ChannelID,
		AccessHash: targetFolder.AccessHash,
	}

	updates, err := c.api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: fromPeer,
		ToPeer:   toPeer,
		ID:       []int{dup.MessageID},
	})
	if err != nil {
		return nil, fmt.Errorf("forward message: %w", err)
	}

	messageID := extractMessageID(updates)

	file, err := c.db.CreateFile(
		targetFolder.ID, fileName, fileSize, mimeType, fileHash,
		messageID, "", nil, true, dup.FileID,
	)
	if err != nil {
		return nil, err
	}

	c.logger.Info("forwarded duplicate file",
		zap.String("name", fileName),
		zap.String("source_file", dup.FileID),
	)
	return file, nil
}

func extractMessageID(updates tg.UpdatesClass) int {
	switch u := updates.(type) {
	case *tg.Updates:
		for _, update := range u.Updates {
			switch upd := update.(type) {
			case *tg.UpdateNewChannelMessage:
				if msg, ok := upd.Message.(*tg.Message); ok {
					return msg.ID
				}
			case *tg.UpdateMessageID:
				return upd.ID
			}
		}
	case *tg.UpdatesCombined:
		for _, update := range u.Updates {
			if upd, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if msg, ok := upd.Message.(*tg.Message); ok {
					return msg.ID
				}
			}
		}
	case *tg.UpdateShortSentMessage:
		return u.ID
	}
	return 0
}

func (c *Client) DeleteFile(ctx context.Context, fileID string) error {
	file, err := c.db.GetFile(fileID)
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}

	folder, err := c.db.GetFolder(file.FolderID)
	if err != nil {
		return fmt.Errorf("get folder: %w", err)
	}

	_, err = c.api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
		Channel: &tg.InputChannel{
			ChannelID:  folder.ChannelID,
			AccessHash: folder.AccessHash,
		},
		ID: []int{file.MessageID},
	})
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return c.db.DeleteFile(fileID)
}

func (c *Client) RenameFile(ctx context.Context, fileID, newName string) error {
	file, err := c.db.GetFile(fileID)
	if err != nil {
		return err
	}

	folder, err := c.db.GetFolder(file.FolderID)
	if err != nil {
		return err
	}

	newMimeType := mime.TypeByExtension(filepath.Ext(newName))
	if newMimeType == "" {
		newMimeType = file.MimeType
	}

	caption := BuildCaption(newName, newMimeType, file.SHA256Hash, file.Size)

	_, err = c.api.MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  folder.ChannelID,
			AccessHash: folder.AccessHash,
		},
		ID:      file.MessageID,
		Message: caption,
	})
	if err != nil {
		c.logger.Warn("failed to edit telegram caption", zap.Error(err))
	}

	return c.db.RenameFile(fileID, newName)
}

// unused import guard
var _ = strconv.Itoa
