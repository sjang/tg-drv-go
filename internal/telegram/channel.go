package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg-drv-go/internal/storage"
)

func (c *Client) CreateChannel(clientCtx context.Context, title string) (*storage.Folder, error) {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return nil, err
	}
	ctx, cancel := mergeContexts(clientCtx, runCtx)
	defer cancel()

	updates, err := c.api.ChannelsCreateChannel(ctx, &tg.ChannelsCreateChannelRequest{
		Title:     title,
		About:     "tg-drv storage folder",
		Broadcast: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	var channel *tg.Channel
	switch u := updates.(type) {
	case *tg.Updates:
		for _, chat := range u.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				channel = ch
				break
			}
		}
	case *tg.UpdatesCombined:
		for _, chat := range u.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				channel = ch
				break
			}
		}
	}

	if channel == nil {
		return nil, fmt.Errorf("channel not found in response")
	}

	folder, err := c.db.CreateFolder(title, channel.ID, channel.AccessHash)
	if err != nil {
		return nil, fmt.Errorf("save folder: %w", err)
	}

	c.logger.Info("created channel/folder",
		zap.String("name", title),
		zap.Int64("channel_id", channel.ID),
	)
	return folder, nil
}

func (c *Client) DeleteChannel(clientCtx context.Context, folderID string) error {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return err
	}
	ctx, cancel := mergeContexts(clientCtx, runCtx)
	defer cancel()

	folder, err := c.db.GetFolder(folderID)
	if err != nil {
		return fmt.Errorf("get folder: %w", err)
	}

	_, err = c.api.ChannelsDeleteChannel(ctx, &tg.InputChannel{
		ChannelID:  folder.ChannelID,
		AccessHash: folder.AccessHash,
	})
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}

	return c.db.DeleteFolder(folderID)
}

func (c *Client) RenameChannel(clientCtx context.Context, folderID, newName string) error {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return err
	}
	ctx, cancel := mergeContexts(clientCtx, runCtx)
	defer cancel()

	folder, err := c.db.GetFolder(folderID)
	if err != nil {
		return err
	}

	_, err = c.api.ChannelsEditTitle(ctx, &tg.ChannelsEditTitleRequest{
		Channel: &tg.InputChannel{
			ChannelID:  folder.ChannelID,
			AccessHash: folder.AccessHash,
		},
		Title: newName,
	})
	if err != nil {
		return fmt.Errorf("rename channel: %w", err)
	}

	return c.db.UpdateFolder(folderID, newName)
}

func (c *Client) SyncChannels(clientCtx context.Context) ([]storage.Folder, error) {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return nil, err
	}
	ctx, cancel := mergeContexts(clientCtx, runCtx)
	defer cancel()

	dialogs, err := c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      500,
	})
	if err != nil {
		return nil, fmt.Errorf("get dialogs: %w", err)
	}

	var channels []*tg.Channel
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		for _, chat := range d.Chats {
			if ch, ok := chat.(*tg.Channel); ok && ch.Broadcast {
				channels = append(channels, ch)
			}
		}
	case *tg.MessagesDialogsSlice:
		for _, chat := range d.Chats {
			if ch, ok := chat.(*tg.Channel); ok && ch.Broadcast {
				channels = append(channels, ch)
			}
		}
	}

	var folders []storage.Folder
	for _, ch := range channels {
		existing, err := c.db.GetFolderByChannelID(ch.ID)
		if err != nil {
			folder, err := c.db.CreateFolder(ch.Title, ch.ID, ch.AccessHash)
			if err != nil {
				c.logger.Warn("failed to sync channel", zap.Int64("id", ch.ID))
				continue
			}
			folders = append(folders, *folder)
		} else {
			folders = append(folders, *existing)
		}
	}

	// Rebuild file index for each synced folder
	for i, folder := range folders {
		count, err := c.RebuildIndex(ctx, folder.ID)
		if err != nil {
			c.logger.Warn("sync: rebuild index failed", zap.String("folder", folder.Name), zap.Error(err))
			continue
		}
		// Re-read folder to get updated file_count/total_size
		updated, err := c.db.GetFolder(folder.ID)
		if err == nil {
			folders[i] = *updated
		}
		c.logger.Info("sync: indexed folder", zap.String("folder", folder.Name), zap.Int("files", count))
	}

	return folders, nil
}

type fileCaption struct {
	Version  int    `json:"v"`
	MimeType string `json:"mime"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	TS       int64  `json:"ts"`
}

func BuildCaption(filename, mimeType, hash string, size int64) string {
	meta := fileCaption{
		Version:  1,
		MimeType: mimeType,
		Hash:     hash,
		Size:     size,
		TS:       time.Now().Unix(),
	}
	data, _ := json.Marshal(meta)
	return filename + "\n" + string(data)
}

func ParseCaption(caption string) (filename, mimeType, hash string, size int64, ok bool) {
	parts := strings.SplitN(caption, "\n", 2)
	if len(parts) != 2 {
		return "", "", "", 0, false
	}
	filename = parts[0]
	var meta fileCaption
	if err := json.Unmarshal([]byte(parts[1]), &meta); err != nil {
		return filename, "", "", 0, false
	}
	return filename, meta.MimeType, meta.Hash, meta.Size, true
}

// RebuildIndex performs incremental sync: fetches only new messages and verifies deletions.
func (c *Client) RebuildIndex(clientCtx context.Context, folderID string) (int, error) {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return 0, err
	}
	ctx, cancel := mergeContexts(clientCtx, runCtx)
	defer cancel()

	folder, err := c.db.GetFolder(folderID)
	if err != nil {
		return 0, err
	}

	inputChannel := &tg.InputChannel{
		ChannelID:  folder.ChannelID,
		AccessHash: folder.AccessHash,
	}
	peer := &tg.InputPeerChannel{
		ChannelID:  folder.ChannelID,
		AccessHash: folder.AccessHash,
	}

	// Phase 1: Incremental — fetch messages newer than the latest known message_id
	maxMsgID, err := c.db.MaxMessageID(folderID)
	if err != nil {
		return 0, fmt.Errorf("get max message_id: %w", err)
	}

	added := 0
	offsetID := 0 // start from newest
	done := false

	for !done {
		history, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     peer,
			Limit:    100,
			OffsetID: offsetID,
		})
		if err != nil {
			return added, fmt.Errorf("get history: %w", err)
		}

		var messages []tg.MessageClass
		switch h := history.(type) {
		case *tg.MessagesMessages:
			messages = h.Messages
		case *tg.MessagesMessagesSlice:
			messages = h.Messages
		case *tg.MessagesChannelMessages:
			messages = h.Messages
		}

		if len(messages) == 0 {
			break
		}

		for _, msgClass := range messages {
			msg, ok := msgClass.(*tg.Message)
			if !ok {
				continue
			}

			// Stop when we reach already-indexed messages
			if msg.ID <= maxMsgID {
				// But skip if this exact message is already in DB
				exists, _ := c.db.FileExistsByMessageID(folderID, msg.ID)
				if exists {
					done = true
					break
				}
			}

			if msg.Media == nil {
				continue
			}

			doc, ok := msg.Media.(*tg.MessageMediaDocument)
			if !ok || doc.Document == nil {
				continue
			}

			document, ok := doc.Document.(*tg.Document)
			if !ok {
				continue
			}

			// Skip if already indexed
			exists, _ := c.db.FileExistsByMessageID(folderID, msg.ID)
			if exists {
				continue
			}

			filename, mimeType, hash, size, parsed := ParseCaption(msg.Message)
			if !parsed {
				for _, attr := range document.Attributes {
					if fa, ok := attr.(*tg.DocumentAttributeFilename); ok {
						filename = fa.FileName
						break
					}
				}
				mimeType = document.MimeType
				size = document.Size
				hash = ""
			}

			if filename == "" {
				filename = fmt.Sprintf("file_%d", msg.ID)
			}

			_, err := c.db.CreateFile(
				folderID, filename, size, mimeType, hash,
				msg.ID, strconv.FormatInt(document.ID, 10),
				nil, false, "",
			)
			if err != nil {
				c.logger.Warn("sync: skip file", zap.Int("msg_id", msg.ID), zap.Error(err))
				continue
			}
			added++
		}

		lastMsg := messages[len(messages)-1]
		switch m := lastMsg.(type) {
		case *tg.Message:
			offsetID = m.ID
		case *tg.MessageService:
			offsetID = m.ID
		default:
			done = true
		}
	}

	// Phase 2: Deletion verification — check if DB files still exist in Telegram
	deleted := 0
	msgIDs, err := c.db.ListMessageIDs(folderID)
	if err != nil {
		c.logger.Warn("sync: failed to list message IDs", zap.Error(err))
	} else if len(msgIDs) > 0 {
		deleted, err = c.verifyDeletions(ctx, inputChannel, folderID, msgIDs)
		if err != nil {
			c.logger.Warn("sync: deletion check failed", zap.Error(err))
		}
	}

	totalFiles, _ := c.db.MaxMessageID(folderID) // just for logging
	_ = totalFiles
	c.logger.Info("sync complete",
		zap.String("folder", folder.Name),
		zap.Int("added", added),
		zap.Int("deleted", deleted),
	)

	// Return total file count
	allMsgIDs, _ := c.db.ListMessageIDs(folderID)
	return len(allMsgIDs), nil
}

// verifyDeletions checks if messages still exist in Telegram, removes DB entries for deleted ones.
func (c *Client) verifyDeletions(ctx context.Context, channel *tg.InputChannel, folderID string, msgIDs []int) (int, error) {
	deleted := 0
	// Process in batches of 100 (Telegram API limit)
	for i := 0; i < len(msgIDs); i += 100 {
		end := i + 100
		if end > len(msgIDs) {
			end = len(msgIDs)
		}
		batch := msgIDs[i:end]

		ids := make([]tg.InputMessageClass, len(batch))
		for j, id := range batch {
			ids[j] = &tg.InputMessageID{ID: id}
		}

		result, err := c.api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: channel,
			ID:      ids,
		})
		if err != nil {
			return deleted, fmt.Errorf("get messages batch: %w", err)
		}

		var messages []tg.MessageClass
		switch m := result.(type) {
		case *tg.MessagesMessages:
			messages = m.Messages
		case *tg.MessagesChannelMessages:
			messages = m.Messages
		}

		// Build set of existing message IDs
		existingIDs := make(map[int]bool)
		for _, msgClass := range messages {
			switch m := msgClass.(type) {
			case *tg.Message:
				existingIDs[m.ID] = true
			case *tg.MessageEmpty:
				// This message was deleted
			}
		}

		// Find deleted message IDs
		var deletedIDs []int
		for _, id := range batch {
			if !existingIDs[id] {
				deletedIDs = append(deletedIDs, id)
			}
		}

		if len(deletedIDs) > 0 {
			n, err := c.db.DeleteFilesByMessageIDs(folderID, deletedIDs)
			if err != nil {
				c.logger.Warn("sync: delete files failed", zap.Error(err))
			}
			deleted += int(n)
		}
	}
	return deleted, nil
}

// FullRebuildIndex clears all files and rescans from scratch (for disaster recovery).
func (c *Client) FullRebuildIndex(ctx context.Context, folderID string) (int, error) {
	_, err := c.db.Exec("DELETE FROM files WHERE folder_id = ?", folderID)
	if err != nil {
		return 0, fmt.Errorf("clear files: %w", err)
	}
	return c.RebuildIndex(ctx, folderID)
}
