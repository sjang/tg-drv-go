package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (db *DB) CreateFolder(name string, channelID, accessHash int64) (*Folder, error) {
	f := &Folder{
		ID:         uuid.New().String(),
		Name:       name,
		ChannelID:  channelID,
		AccessHash: accessHash,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	_, err := db.Exec(
		`INSERT INTO folders (id, name, channel_id, access_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		f.ID, f.Name, f.ChannelID, f.AccessHash, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	return f, nil
}

func (db *DB) ListFolders() ([]Folder, error) {
	rows, err := db.Query(`
		SELECT f.id, f.name, f.channel_id, f.access_hash, f.created_at, f.updated_at,
		       COALESCE(s.cnt, 0), COALESCE(s.total, 0)
		FROM folders f
		LEFT JOIN (
			SELECT folder_id, COUNT(*) as cnt, SUM(size) as total
			FROM files GROUP BY folder_id
		) s ON f.id = s.folder_id
		ORDER BY f.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.ChannelID, &f.AccessHash,
			&f.CreatedAt, &f.UpdatedAt, &f.FileCount, &f.TotalSize); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (db *DB) GetFolder(id string) (*Folder, error) {
	var f Folder
	err := db.QueryRow(
		`SELECT id, name, channel_id, access_hash, created_at, updated_at FROM folders WHERE id = ?`, id,
	).Scan(&f.ID, &f.Name, &f.ChannelID, &f.AccessHash, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (db *DB) GetFolderByChannelID(channelID int64) (*Folder, error) {
	var f Folder
	err := db.QueryRow(
		`SELECT id, name, channel_id, access_hash, created_at, updated_at FROM folders WHERE channel_id = ?`, channelID,
	).Scan(&f.ID, &f.Name, &f.ChannelID, &f.AccessHash, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (db *DB) UpdateFolder(id, name string) error {
	_, err := db.Exec(
		`UPDATE folders SET name = ?, updated_at = ? WHERE id = ?`,
		name, time.Now(), id,
	)
	return err
}

func (db *DB) DeleteFolder(id string) error {
	_, err := db.Exec(`DELETE FROM folders WHERE id = ?`, id)
	return err
}
