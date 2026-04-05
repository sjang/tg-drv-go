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

func (c *Client) CreateChannel(ctx context.Context, title string) (*storage.Folder, error) {
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

func (c *Client) DeleteChannel(ctx context.Context, folderID string) error {
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

func (c *Client) RenameChannel(ctx context.Context, folderID, newName string) error {
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

func (c *Client) SyncChannels(ctx context.Context) ([]storage.Folder, error) {
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

func (c *Client) RebuildIndex(ctx context.Context, folderID string) (int, error) {
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

	// Delete existing files for this folder
	_, err = c.db.Exec("DELETE FROM files WHERE folder_id = ?", folderID)
	if err != nil {
		return 0, fmt.Errorf("clear files: %w", err)
	}

	count := 0
	offsetID := 0

	for {
		history, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     peer,
			Limit:    100,
			OffsetID: offsetID,
		})
		if err != nil {
			return count, fmt.Errorf("get history: %w", err)
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
			if !ok || msg.Media == nil {
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

			filename, mimeType, hash, size, parsed := ParseCaption(msg.Message)
			if !parsed {
				// Try to extract from document attributes
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
				c.logger.Warn("rebuild: skip file", zap.Int("msg_id", msg.ID), zap.Error(err))
				continue
			}
			count++
		}

		lastMsg := messages[len(messages)-1]
		switch m := lastMsg.(type) {
		case *tg.Message:
			offsetID = m.ID
		case *tg.MessageService:
			offsetID = m.ID
		default:
			break
		}
	}

	_ = inputChannel
	c.logger.Info("rebuilt index", zap.String("folder", folder.Name), zap.Int("files", count))
	return count, nil
}
