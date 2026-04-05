package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"

	"tg-drv-go/internal/api"
	"tg-drv-go/internal/config"
	"tg-drv-go/internal/storage"
	"tg-drv-go/internal/telegram"
)

type App struct {
	ctx       context.Context
	cfg       *config.Config
	db        *storage.DB
	tg        *telegram.Client
	apiServer *api.Server
	logger    *zap.Logger
	cancel    context.CancelFunc
}

func NewApp() *App {
	logger, _ := zap.NewDevelopment()
	return &App{
		logger: logger,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load config
	configPath := filepath.Join(config.DefaultDataDir(), "config.json")
	cfg, err := config.Load(configPath)
	if err != nil {
		a.logger.Fatal("load config", zap.Error(err))
	}
	a.cfg = cfg

	if err := cfg.EnsureDataDir(); err != nil {
		a.logger.Fatal("ensure data dir", zap.Error(err))
	}

	// Open DB
	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		a.logger.Fatal("open db", zap.Error(err))
	}
	a.db = db

	// Initialize Telegram client
	if cfg.TelegramAPIID == 0 || cfg.TelegramAPIHash == "" {
		a.logger.Warn("Telegram API credentials not configured. Set them in " + configPath)
		return
	}

	a.tg = telegram.NewClient(cfg.TelegramAPIID, cfg.TelegramAPIHash, db, a.logger)

	// Run Telegram client in background
	tgCtx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	go func() {
		if err := a.tg.Run(tgCtx); err != nil {
			a.logger.Error("telegram client stopped", zap.Error(err))
		}
	}()

	// Start API server in background
	a.apiServer = api.NewServer(a.tg, cfg.HTTPPort, a.logger)
	go func() {
		if err := a.apiServer.Start(); err != nil {
			a.logger.Error("api server stopped", zap.Error(err))
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.apiServer != nil {
		a.apiServer.Shutdown(ctx)
	}
	if a.db != nil {
		a.db.Close()
	}
	a.logger.Sync()
}

// --- Wails-bound methods (called from frontend) ---

func (a *App) GetAuthStatus() telegram.AuthStatus {
	if a.tg == nil {
		return telegram.AuthStatus{Authenticated: false}
	}
	return a.tg.GetAuthStatus()
}

func (a *App) SendAuthCode(phone string) error {
	if a.tg == nil {
		return fmt.Errorf("telegram client not initialized")
	}
	return a.tg.SendCode(a.ctx, phone)
}

func (a *App) VerifyAuthCode(code string) error {
	if a.tg == nil {
		return fmt.Errorf("telegram client not initialized")
	}
	return a.tg.VerifyCode(a.ctx, code)
}

func (a *App) VerifyPassword(password string) error {
	if a.tg == nil {
		return fmt.Errorf("telegram client not initialized")
	}
	return a.tg.VerifyPassword(a.ctx, password)
}

func (a *App) GetFolders() ([]storage.Folder, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a.db.ListFolders()
}

func (a *App) CreateFolder(name string) (*storage.Folder, error) {
	if a.tg == nil {
		return nil, fmt.Errorf("telegram client not initialized")
	}
	return a.tg.CreateChannel(a.ctx, name)
}

func (a *App) RenameFolder(id, name string) error {
	if a.tg == nil {
		return fmt.Errorf("telegram client not initialized")
	}
	return a.tg.RenameChannel(a.ctx, id, name)
}

func (a *App) DeleteFolder(id string) error {
	if a.tg == nil {
		return fmt.Errorf("telegram client not initialized")
	}
	return a.tg.DeleteChannel(a.ctx, id)
}

func (a *App) GetFiles(folderID string) ([]storage.File, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a.db.ListFiles(folderID)
}

func (a *App) UploadFile(folderID string) (*storage.File, error) {
	if a.tg == nil {
		return nil, fmt.Errorf("telegram client not initialized")
	}

	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select file to upload",
	})
	if err != nil || filePath == "" {
		return nil, fmt.Errorf("no file selected")
	}

	return a.tg.UploadFile(a.ctx, folderID, filePath, func(p telegram.UploadProgress) {
		runtime.EventsEmit(a.ctx, "upload:progress", p)
	})
}

func (a *App) UploadFilePath(folderID, filePath string) (*storage.File, error) {
	if a.tg == nil {
		return nil, fmt.Errorf("telegram client not initialized")
	}
	return a.tg.UploadFile(a.ctx, folderID, filePath, func(p telegram.UploadProgress) {
		runtime.EventsEmit(a.ctx, "upload:progress", p)
	})
}

func (a *App) DeleteFile(fileID string) error {
	if a.tg == nil {
		return fmt.Errorf("telegram client not initialized")
	}
	return a.tg.DeleteFile(a.ctx, fileID)
}

func (a *App) RenameFile(fileID, newName string) error {
	if a.tg == nil {
		return fmt.Errorf("telegram client not initialized")
	}
	return a.tg.RenameFile(a.ctx, fileID, newName)
}

func (a *App) GetStreamURL(fileID string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/api/files/%s/stream", a.cfg.HTTPPort, fileID)
}

func (a *App) GetThumbnailURL(fileID string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/api/files/%s/thumbnail", a.cfg.HTTPPort, fileID)
}

func (a *App) RebuildIndex(folderID string) (int, error) {
	if a.tg == nil {
		return 0, fmt.Errorf("telegram client not initialized")
	}
	return a.tg.RebuildIndex(a.ctx, folderID)
}

func (a *App) SyncFolders() ([]storage.Folder, error) {
	if a.tg == nil {
		return nil, fmt.Errorf("telegram client not initialized")
	}
	return a.tg.SyncChannels(a.ctx)
}

func (a *App) GetAPIPort() int {
	return a.cfg.HTTPPort
}

func (a *App) IsConfigured() bool {
	return a.cfg != nil && a.cfg.TelegramAPIID != 0 && a.cfg.TelegramAPIHash != ""
}

func (a *App) SaveConfig(apiID int, apiHash string) error {
	a.cfg.TelegramAPIID = apiID
	a.cfg.TelegramAPIHash = apiHash
	configPath := filepath.Join(config.DefaultDataDir(), "config.json")
	return a.cfg.Save(configPath)
}

// unused import guard
var _ = os.Remove
