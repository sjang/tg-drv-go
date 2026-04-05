package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (db *DB) CreateFile(folderID, name string, size int64, mimeType, sha256Hash string, messageID int, telegramFileID string, thumbnail []byte, isDuplicate bool, sourceFileID string) (*File, error) {
	f := &File{
		ID:             uuid.New().String(),
		FolderID:       folderID,
		Name:           name,
		Size:           size,
		MimeType:       mimeType,
		SHA256Hash:     sha256Hash,
		MessageID:      messageID,
		TelegramFileID: telegramFileID,
		Thumbnail:      thumbnail,
		HasThumbnail:   len(thumbnail) > 0,
		UploadDate:     time.Now(),
		IsDuplicate:    isDuplicate,
		SourceFileID:   sourceFileID,
	}
	_, err := db.Exec(
		`INSERT INTO files (id, folder_id, name, size, mime_type, sha256_hash, message_id, telegram_file_id, thumbnail, upload_date, is_duplicate, source_file_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.FolderID, f.Name, f.Size, f.MimeType, f.SHA256Hash,
		f.MessageID, f.TelegramFileID, f.Thumbnail, f.UploadDate,
		f.IsDuplicate, f.SourceFileID,
	)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	return f, nil
}

func (db *DB) ListFiles(folderID string) ([]File, error) {
	rows, err := db.Query(`
		SELECT id, folder_id, name, size, mime_type, sha256_hash, message_id,
		       telegram_file_id, thumbnail IS NOT NULL AND length(thumbnail) > 0,
		       upload_date, is_duplicate, COALESCE(source_file_id, '')
		FROM files WHERE folder_id = ?
		ORDER BY upload_date DESC`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.FolderID, &f.Name, &f.Size, &f.MimeType,
			&f.SHA256Hash, &f.MessageID, &f.TelegramFileID, &f.HasThumbnail,
			&f.UploadDate, &f.IsDuplicate, &f.SourceFileID); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (db *DB) GetFile(id string) (*File, error) {
	var f File
	err := db.QueryRow(`
		SELECT id, folder_id, name, size, mime_type, sha256_hash, message_id,
		       telegram_file_id, thumbnail, upload_date, is_duplicate, COALESCE(source_file_id, '')
		FROM files WHERE id = ?`, id,
	).Scan(&f.ID, &f.FolderID, &f.Name, &f.Size, &f.MimeType, &f.SHA256Hash,
		&f.MessageID, &f.TelegramFileID, &f.Thumbnail, &f.UploadDate,
		&f.IsDuplicate, &f.SourceFileID)
	if err != nil {
		return nil, err
	}
	f.HasThumbnail = len(f.Thumbnail) > 0
	return &f, nil
}

func (db *DB) GetFileThumbnail(id string) ([]byte, error) {
	var thumb []byte
	err := db.QueryRow(`SELECT thumbnail FROM files WHERE id = ?`, id).Scan(&thumb)
	if err != nil {
		return nil, err
	}
	return thumb, nil
}

func (db *DB) RenameFile(id, newName string) error {
	_, err := db.Exec(`UPDATE files SET name = ? WHERE id = ?`, newName, id)
	return err
}

func (db *DB) DeleteFile(id string) error {
	_, err := db.Exec(`DELETE FROM files WHERE id = ?`, id)
	return err
}

func (db *DB) MoveFile(id, targetFolderID string) error {
	_, err := db.Exec(`UPDATE files SET folder_id = ? WHERE id = ?`, targetFolderID, id)
	return err
}

func (db *DB) UpdateFileThumbnail(id string, thumbnail []byte) error {
	_, err := db.Exec(`UPDATE files SET thumbnail = ? WHERE id = ?`, thumbnail, id)
	return err
}
