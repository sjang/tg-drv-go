package storage

import "time"

type Folder struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ChannelID  int64     `json:"channel_id"`
	AccessHash int64     `json:"access_hash"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	FileCount  int       `json:"file_count,omitempty"`
	TotalSize  int64     `json:"total_size,omitempty"`
}

type File struct {
	ID             string    `json:"id"`
	FolderID       string    `json:"folder_id"`
	Name           string    `json:"name"`
	Size           int64     `json:"size"`
	MimeType       string    `json:"mime_type"`
	SHA256Hash     string    `json:"sha256_hash"`
	MessageID      int       `json:"message_id"`
	TelegramFileID string    `json:"telegram_file_id,omitempty"`
	Thumbnail      []byte    `json:"-"`
	HasThumbnail   bool      `json:"has_thumbnail"`
	UploadDate     time.Time `json:"upload_date"`
	IsDuplicate    bool      `json:"is_duplicate"`
	SourceFileID   string    `json:"source_file_id,omitempty"`
}

type UploadQueueItem struct {
	ID           string    `json:"id"`
	FolderID     string    `json:"folder_id"`
	LocalPath    string    `json:"local_path"`
	FileName     string    `json:"file_name"`
	TotalSize    int64     `json:"total_size"`
	UploadedSize int64     `json:"uploaded_size"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
