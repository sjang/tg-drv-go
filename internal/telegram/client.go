package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg-drv-go/internal/storage"
)

type Client struct {
	client     *telegram.Client
	api        *tg.Client
	uploader   *uploader.Uploader
	downloader *downloader.Downloader
	db         *storage.DB
	logger     *zap.Logger

	mu          sync.RWMutex
	ready       bool
	readyCh     chan struct{}
	self        *tg.User
	authPending *authState
}

type authState struct {
	phone    string
	codeHash string
}

type AuthStatus struct {
	Authenticated bool    `json:"authenticated"`
	PhoneNumber   string  `json:"phone_number,omitempty"`
	FirstName     string  `json:"first_name,omitempty"`
	LastName      string  `json:"last_name,omitempty"`
	Username      string  `json:"username,omitempty"`
}

func NewClient(apiID int, apiHash string, db *storage.DB, logger *zap.Logger) *Client {
	sessionStore := NewSQLiteSessionStorage(db.DB)

	waiter := floodwait.NewWaiter()

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: sessionStore,
		Logger:         logger.Named("td"),
		Middlewares: []telegram.Middleware{
			waiter,
		},
	})

	return &Client{
		client:  client,
		db:      db,
		logger:  logger,
		readyCh: make(chan struct{}),
	}
}

func (c *Client) Run(ctx context.Context) error {
	return c.client.Run(ctx, func(ctx context.Context) error {
		c.mu.Lock()
		c.api = c.client.API()
		c.uploader = uploader.NewUploader(c.api).WithThreads(4)
		c.downloader = downloader.NewDownloader()
		c.mu.Unlock()

		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("auth status: %w", err)
		}

		if status.Authorized {
			c.mu.Lock()
			c.self = status.User
			c.ready = true
			c.mu.Unlock()
			close(c.readyCh)
			c.logger.Info("authenticated", zap.String("user", status.User.FirstName))
		} else {
			c.mu.Lock()
			c.ready = false
			c.mu.Unlock()
			close(c.readyCh)
			c.logger.Info("not authenticated, waiting for login")
		}

		<-ctx.Done()
		return ctx.Err()
	})
}

func (c *Client) WaitReady(ctx context.Context) error {
	select {
	case <-c.readyCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready && c.self != nil
}

func (c *Client) GetAuthStatus() AuthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.self == nil {
		return AuthStatus{Authenticated: false}
	}
	return AuthStatus{
		Authenticated: true,
		PhoneNumber:   c.self.Phone,
		FirstName:     c.self.FirstName,
		LastName:      c.self.LastName,
		Username:      c.self.Username,
	}
}

func (c *Client) SendCode(ctx context.Context, phone string) error {
	sentCode, err := c.client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
	if err != nil {
		return fmt.Errorf("send code: %w", err)
	}

	codeHash := ""
	switch v := sentCode.(type) {
	case *tg.AuthSentCode:
		codeHash = v.PhoneCodeHash
	default:
		return fmt.Errorf("unexpected sent code type: %T", v)
	}

	c.mu.Lock()
	c.authPending = &authState{phone: phone, codeHash: codeHash}
	c.mu.Unlock()

	return nil
}

func (c *Client) VerifyCode(ctx context.Context, code string) error {
	c.mu.RLock()
	pending := c.authPending
	c.mu.RUnlock()

	if pending == nil {
		return fmt.Errorf("no pending auth, call SendCode first")
	}

	result, err := c.api.AuthSignIn(ctx, &tg.AuthSignInRequest{
		PhoneNumber:   pending.phone,
		PhoneCodeHash: pending.codeHash,
		PhoneCode:     code,
	})
	if err != nil {
		return fmt.Errorf("sign in: %w", err)
	}

	authResult, ok := result.(*tg.AuthAuthorization)
	if !ok {
		return fmt.Errorf("unexpected auth result type: %T", result)
	}

	user, ok := authResult.User.(*tg.User)
	if !ok {
		return fmt.Errorf("unexpected user type")
	}

	c.mu.Lock()
	c.self = user
	c.ready = true
	c.authPending = nil
	c.mu.Unlock()

	c.logger.Info("login successful", zap.String("user", user.FirstName))
	return nil
}

func (c *Client) VerifyPassword(ctx context.Context, password string) error {
	_, err := c.client.Auth().Password(ctx, password)
	if err != nil {
		return fmt.Errorf("2fa: %w", err)
	}

	status, err := c.client.Auth().Status(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.self = status.User
	c.ready = true
	c.authPending = nil
	c.mu.Unlock()

	return nil
}

func (c *Client) API() *tg.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.api
}

func (c *Client) Uploader() *uploader.Uploader {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.uploader
}

func (c *Client) Downloader() *downloader.Downloader {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.downloader
}

func (c *Client) Storage() *storage.DB {
	return c.db
}
