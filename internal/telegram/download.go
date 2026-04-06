package telegram

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

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

// fileLocationCache caches FileLocation lookups to avoid repeated Telegram API calls
// during video streaming where the player issues many Range requests.
type fileLocationCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedLocation
}

type cachedLocation struct {
	loc       *FileLocation
	expiresAt time.Time
}

const fileLocationCacheTTL = 5 * time.Minute

func newFileLocationCache() *fileLocationCache {
	return &fileLocationCache{
		entries: make(map[string]*cachedLocation),
	}
}

func (c *fileLocationCache) get(fileID string) (*FileLocation, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[fileID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.loc, true
}

func (c *fileLocationCache) set(fileID string, loc *FileLocation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[fileID] = &cachedLocation{
		loc:       loc,
		expiresAt: time.Now().Add(fileLocationCacheTTL),
	}
}

func (c *Client) GetFileLocation(ctx context.Context, fileID string) (*FileLocation, error) {
	// Check cache first to avoid Telegram API round-trip on repeated Range requests
	if cached, ok := c.locCache.get(fileID); ok {
		return cached, nil
	}

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

	loc := &FileLocation{
		InputLocation: &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
		},
		Size:     doc.Size,
		MimeType: file.MimeType,
		FileName: file.Name,
	}

	// Cache the resolved location for subsequent Range requests
	c.locCache.set(fileID, loc)

	return loc, nil
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

// chunkResult holds the result of a prefetched chunk.
type chunkResult struct {
	data   []byte
	offset int64
	eof    bool
	err    error
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

	// Prefetch: start fetching the next chunk while processing the current one
	fetchChunk := func(off int64) <-chan chunkResult {
		ch := make(chan chunkResult, 1)
		go func() {
			result, err := c.api.UploadGetFile(runCtx, &tg.UploadGetFileRequest{
				Location: loc.InputLocation,
				Offset:   off,
				Limit:    int(chunkSize),
			})
			if err != nil {
				ch <- chunkResult{err: fmt.Errorf("get file chunk at offset %d: %w", off, err)}
				return
			}
			switch f := result.(type) {
			case *tg.UploadFile:
				ch <- chunkResult{
					data:   f.Bytes,
					offset: off,
					eof:    len(f.Bytes) < int(chunkSize),
				}
			case *tg.UploadFileCDNRedirect:
				ch <- chunkResult{err: fmt.Errorf("CDN redirect not supported")}
			}
		}()
		return ch
	}

	// Start first fetch
	pending := fetchChunk(currentOffset)

	for remaining > 0 {
		res := <-pending
		if res.err != nil {
			return res.err
		}

		// Start prefetching next chunk before processing current one
		nextOffset := currentOffset + chunkSize
		var nextPending <-chan chunkResult
		if remaining > 0 && !res.eof {
			nextPending = fetchChunk(nextOffset)
		}

		data := res.data
		if skip > 0 {
			if int64(len(data)) <= skip {
				skip -= int64(len(data))
				currentOffset += int64(len(data))
				if nextPending != nil {
					pending = nextPending
				}
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

		if res.eof {
			return nil
		}

		if nextPending != nil {
			pending = nextPending
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
