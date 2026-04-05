package telegram

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gotd/td/session"
)

type SQLiteSessionStorage struct {
	db *sql.DB
}

func NewSQLiteSessionStorage(db *sql.DB) *SQLiteSessionStorage {
	return &SQLiteSessionStorage{db: db}
}

func (s *SQLiteSessionStorage) LoadSession(_ context.Context) ([]byte, error) {
	var data []byte
	err := s.db.QueryRow("SELECT session_data FROM tg_sessions WHERE id = 1").Scan(&data)
	if err == sql.ErrNoRows {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	return data, nil
}

func (s *SQLiteSessionStorage) StoreSession(_ context.Context, data []byte) error {
	_, err := s.db.Exec(
		`INSERT INTO tg_sessions (id, session_data, updated_at) VALUES (1, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET session_data = excluded.session_data, updated_at = CURRENT_TIMESTAMP`,
		data,
	)
	if err != nil {
		return fmt.Errorf("store session: %w", err)
	}
	return nil
}
