package telegram

import (
	"context"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// Bridge è il wrapper attorno al client gotd/td.
// Gestisce connessione, session e comunicazione con il TUI via callback.
// I campi callback sono raggruppati in BridgeCallbacks (bridge_callbacks.go).
type Bridge struct {
	client *telegram.Client
	api    *tg.Client
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc

	BridgeCallbacks
}

// NewBridge crea un nuovo bridge Telegram.
func NewBridge(cfg Config) *Bridge {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bridge{cfg: cfg, ctx: ctx, cancel: cancel}
}

// Start avvia il client in background. Non bloccante.
func (b *Bridge) Start() {
	dispatcher := tg.NewUpdateDispatcher()
	b.setupUpdates(&dispatcher)
	b.client = telegram.NewClient(b.cfg.AppID, b.cfg.AppHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: "session.json"},
		UpdateHandler:  dispatcher,
	})
	go b.run()
}

// Stop arresta il client.
func (b *Bridge) Stop() {
	b.cancel()
}

func (b *Bridge) run() {
	err := b.client.Run(b.ctx, func(ctx context.Context) error {
		b.api = b.client.API()
		return b.handleConnection(ctx)
	})
	if err != nil && b.ctx.Err() == nil && b.OnDisconnected != nil {
		b.OnDisconnected(err)
	}
}

func (b *Bridge) handleConnection(ctx context.Context) error {
	status, err := b.client.Auth().Status(ctx)
	if err != nil {
		return err
	}

	if !status.Authorized {
		if b.OnAuthRequired != nil {
			b.OnAuthRequired()
		}
		// Poll fino a che l'auth non è completata dall'UI
		return b.waitForAuth(ctx)
	}

	if b.OnConnected != nil {
		b.OnConnected()
	}

	b.startHealthMonitor(ctx)

	<-ctx.Done()
	return ctx.Err()
}

func (b *Bridge) waitForAuth(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := b.client.Auth().Status(ctx)
			if err != nil {
				continue
			}
			if status.Authorized {
				if b.OnConnected != nil {
					b.OnConnected()
				}
				<-ctx.Done()
				return ctx.Err()
			}
		}
	}
}

// API restituisce il client API raw. Nil se non connesso.
func (b *Bridge) API() *tg.Client {
	return b.api
}

// Client restituisce il client telegram per operazioni auth.
func (b *Bridge) Client() *telegram.Client {
	return b.client
}
