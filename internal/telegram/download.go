package telegram

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/gotd/td/tg"
)

type DownloadProgress struct {
	FileID     string  `json:"file_id"`
	FileName   string  `json:"file_name"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Percent    float64 `json:"percent"`
}

type progressWriter struct {
	w          io.Writer
	fileID     string
	fileName   string
	total      int64
	written    int64
	onProgress func(DownloadProgress)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	pw.written += int64(n)
	if pw.onProgress != nil && pw.total > 0 {
		pw.onProgress(DownloadProgress{
			FileID:     pw.fileID,
			FileName:   pw.fileName,
			Downloaded: pw.written,
			Total:      pw.total,
			Percent:    float64(pw.written) / float64(pw.total) * 100,
		})
	}
	return n, err
}

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

	runCtx, err := c.getRunCtx()
	if err != nil {
		return nil, err
	}

	msgs, err := c.api.ChannelsGetMessages(runCtx, &tg.ChannelsGetMessagesRequest{
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

func (c *Client) DownloadFile(_ context.Context, fileID string, w io.Writer) error {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return err
	}

	loc, err := c.GetFileLocation(runCtx, fileID)
	if err != nil {
		return err
	}

	_, err = c.downloader.Download(c.api, loc.InputLocation).Stream(runCtx, w)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	return nil
}

func (c *Client) DownloadFileWithProgress(_ context.Context, fileID string, w io.Writer, onProgress func(DownloadProgress)) error {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return err
	}

	loc, err := c.GetFileLocation(runCtx, fileID)
	if err != nil {
		return err
	}

	file, err := c.db.GetFile(fileID)
	if err != nil {
		return err
	}

	pw := &progressWriter{
		w:          w,
		fileID:     fileID,
		fileName:   file.Name,
		total:      loc.Size,
		onProgress: onProgress,
	}

	_, err = c.downloader.Download(c.api, loc.InputLocation).Stream(runCtx, pw)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	return nil
}

func (c *Client) DownloadRange(_ context.Context, fileID string, w io.Writer, offset, length int64) error {
	runCtx, err := c.getRunCtx()
	if err != nil {
		return err
	}

	loc, err := c.GetFileLocation(runCtx, fileID)
	if err != nil {
		return err
	}

	const chunkSize int64 = 1048576 // 1MB - Telegram requires offset % limit == 0

	// Align offset down to chunk boundary
	alignedOffset := (offset / chunkSize) * chunkSize
	skip := offset - alignedOffset
	remaining := length
	currentOffset := alignedOffset

	for remaining > 0 {
		result, err := c.api.UploadGetFile(runCtx, &tg.UploadGetFileRequest{
			Location: loc.InputLocation,
			Offset:   currentOffset,
			Limit:    int(chunkSize),
		})
		if err != nil {
			return fmt.Errorf("get file chunk at offset %d: %w", currentOffset, err)
		}

		switch f := result.(type) {
		case *tg.UploadFile:
			data := f.Bytes
			// Skip leading bytes for the first chunk if offset wasn't aligned
			if skip > 0 {
				if int64(len(data)) <= skip {
					skip -= int64(len(data))
					currentOffset += int64(len(data))
					continue
				}
				data = data[skip:]
				skip = 0
			}
			if int64(len(data)) > remaining {
				data = data[:remaining]
			}
			n, err := w.Write(data)
			if err != nil {
				return err
			}
			remaining -= int64(n)
			currentOffset += chunkSize
			if len(f.Bytes) < int(chunkSize) {
				return nil // EOF
			}
		case *tg.UploadFileCDNRedirect:
			return fmt.Errorf("CDN redirect not supported")
		}
	}
	return nil
}

// alignChunkSize rounds up to the nearest power-of-2 multiple of 4096.
// Telegram's upload.getFile requires limit to be one of:
// 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576
func alignChunkSize(size int64) int64 {
	if size <= 4096 {
		return 4096
	}
	if size > 1048576 {
		return 1048576
	}
	// Round up to next power of 2
	v := size - 1
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	return v + 1
}

// unused import guard
var _ = strconv.Itoa
