package storage

import "database/sql"

type DuplicateInfo struct {
	FileID    string
	FolderID  string
	MessageID int
	ChannelID int64
}

func (db *DB) FindDuplicate(sha256Hash string) (*DuplicateInfo, error) {
	var info DuplicateInfo
	err := db.QueryRow(`
		SELECT f.id, f.folder_id, f.message_id, fo.channel_id
		FROM files f
		JOIN folders fo ON f.folder_id = fo.id
		WHERE f.sha256_hash = ?
		LIMIT 1`, sha256Hash,
	).Scan(&info.FileID, &info.FolderID, &info.MessageID, &info.ChannelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (db *DB) FileExistsInFolder(folderID, sha256Hash string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM files WHERE folder_id = ? AND sha256_hash = ?`,
		folderID, sha256Hash).Scan(&count)
	return count > 0, err
}
