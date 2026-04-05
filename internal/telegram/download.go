package telegram

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/gotd/td/tg"
)

type FileLocation struct {
	InputLocation tg.InputFileLocationClass
	Size          int64
	MimeType      string
	FileName      string
}

func (c *Client) GetFileLocation(ctx context.Context, fileID string) (*FileLocation, error) {
	file, err := c.db.GetFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}

	folder, err := c.db.GetFolder(file.FolderID)
	if err != nil {
		return nil, fmt.Errorf("get folder: %w", err)
	}

	// Fetch the message to get the document
	msgs, err := c.api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: &tg.InputChannel{
			ChannelID:  folder.ChannelID,
			AccessHash: folder.AccessHash,
		},
		ID: []tg.InputMessageClass{&tg.InputMessageID{ID: file.MessageID}},
	})
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	var messages []tg.MessageClass
	switch m := msgs.(type) {
	case *tg.MessagesMessages:
		messages = m.Messages
	case *tg.MessagesChannelMessages:
		messages = m.Messages
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("message not found")
	}

	msg, ok := messages[0].(*tg.Message)
	if !ok || msg.Media == nil {
		return nil, fmt.Errorf("message has no media")
	}

	mediaDoc, ok := msg.Media.(*tg.MessageMediaDocument)
	if !ok || mediaDoc.Document == nil {
		return nil, fmt.Errorf("message has no document")
	}

	doc, ok := mediaDoc.Document.(*tg.Document)
	if !ok {
		return nil, fmt.Errorf("unexpected document type")
	}

	return &FileLocation{
		InputLocation: &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
		},
		Size:     doc.Size,
		MimeType: file.MimeType,
		FileName: file.Name,
	}, nil
}

func (c *Client) DownloadFile(ctx context.Context, fileID string, w io.Writer) error {
	loc, err := c.GetFileLocation(ctx, fileID)
	if err != nil {
		return err
	}

	_, err = c.downloader.Download(c.api, loc.InputLocation).Stream(ctx, w)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	return nil
}

func (c *Client) DownloadRange(ctx context.Context, fileID string, w io.Writer, offset, limit int64) error {
	loc, err := c.GetFileLocation(ctx, fileID)
	if err != nil {
		return err
	}

	// For range requests, we need to use the raw API to download chunks
	// gotd/td downloader doesn't directly support offset-based streaming
	// so we use GetFile API calls with offset
	chunkSize := int64(1024 * 1024) // 1MB chunks
	remaining := limit

	for remaining > 0 {
		reqLimit := chunkSize
		if remaining < reqLimit {
			reqLimit = remaining
		}
		// Ensure limit is a valid power-of-2 multiple for Telegram
		reqLimit = alignChunkSize(reqLimit)

		result, err := c.api.UploadGetFile(ctx, &tg.UploadGetFileRequest{
			Location: loc.InputLocation,
			Offset:   offset,
			Limit:    int(reqLimit),
		})
		if err != nil {
			return fmt.Errorf("get file chunk: %w", err)
		}

		switch f := result.(type) {
		case *tg.UploadFile:
			data := f.Bytes
			if int64(len(data)) > remaining {
				data = data[:remaining]
			}
			n, err := w.Write(data)
			if err != nil {
				return err
			}
			offset += int64(n)
			remaining -= int64(n)
			if len(f.Bytes) < int(reqLimit) {
				// EOF
				return nil
			}
		case *tg.UploadFileCDNRedirect:
			return fmt.Errorf("CDN redirect not supported")
		}
	}
	return nil
}

func alignChunkSize(size int64) int64 {
	// Telegram requires chunk size to be a multiple of 4096
	// and between 4096 and 1048576
	if size < 4096 {
		return 4096
	}
	if size > 1048576 {
		return 1048576
	}
	// Round up to nearest 4096
	return ((size + 4095) / 4096) * 4096
}

// unused import guard
var _ = strconv.Itoa
