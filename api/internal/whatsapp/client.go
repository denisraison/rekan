package whatsapp

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
)

// EventHandler is called for each incoming WhatsApp event.
type EventHandler func(evt any)

// Client wraps a whatsmeow client with session management and QR pairing.
type Client struct {
	wac       *whatsmeow.Client
	container *sqlstore.Container

	mu     sync.RWMutex
	qrCode string // current QR code string, empty when connected or expired
}

// Status represents the current WhatsApp connection state.
type Status struct {
	Connected bool   `json:"connected"`
	QRCode    string `json:"qr,omitempty"`
}

// New creates a whatsmeow client backed by a SQLite session store.
// The client is not connected yet; call Connect to start.
func New(ctx context.Context, dbPath string) (*Client, error) {
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_pragma=journal_mode(WAL)", dbPath)
	container, err := sqlstore.New(ctx, "sqlite", dsn, nil)
	if err != nil {
		return nil, fmt.Errorf("whatsapp store: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp device: %w", err)
	}

	wac := whatsmeow.NewClient(device, nil)

	return &Client{
		wac:       wac,
		container: container,
	}, nil
}

// Connect starts the WhatsApp connection. If no session exists, it begins
// QR code pairing (poll Status() for QR codes). If a session exists, it
// reconnects automatically.
func (c *Client) Connect(ctx context.Context) error {
	if c.wac.Store.ID == nil {
		// No session: start QR pairing flow
		qrChan, err := c.wac.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("whatsapp qr channel: %w", err)
		}
		if err := c.wac.Connect(); err != nil {
			return fmt.Errorf("whatsapp connect: %w", err)
		}
		go c.handleQR(qrChan)
		return nil
	}

	// Existing session: reconnect
	if err := c.wac.Connect(); err != nil {
		return fmt.Errorf("whatsapp connect: %w", err)
	}
	return nil
}

// Disconnect gracefully disconnects from WhatsApp.
func (c *Client) Disconnect() {
	c.wac.Disconnect()
}

// Status returns the current connection state and QR code (if pairing).
func (c *Client) Status() Status {
	c.mu.RLock()
	qr := c.qrCode
	c.mu.RUnlock()
	return Status{
		Connected: c.wac.IsConnected(),
		QRCode:    qr,
	}
}

// AddEventHandler registers a handler for WhatsApp events (messages, receipts, etc.).
func (c *Client) AddEventHandler(handler EventHandler) {
	c.wac.AddEventHandler(func(evt any) {
		handler(evt)
	})
}

// SendMessage sends a message to the given JID.
func (c *Client) SendMessage(ctx context.Context, to types.JID, msg *waE2E.Message) (whatsmeow.SendResponse, error) {
	return c.wac.SendMessage(ctx, to, msg)
}

// SendChatPresence sends a typing indicator to the given chat.
func (c *Client) SendChatPresence(ctx context.Context, jid types.JID, state types.ChatPresence, media types.ChatPresenceMedia) error {
	return c.wac.SendChatPresence(ctx, jid, state, media)
}

// Download downloads media from a message.
func (c *Client) Download(ctx context.Context, msg whatsmeow.DownloadableMessage) ([]byte, error) {
	return c.wac.Download(ctx, msg)
}

func (c *Client) handleQR(ch <-chan whatsmeow.QRChannelItem) {
	for evt := range ch {
		c.mu.Lock()
		switch evt.Event {
		case "code":
			c.qrCode = evt.Code
		default:
			// success, timeout, error — clear QR
			c.qrCode = ""
		}
		c.mu.Unlock()
		log.Printf("whatsapp qr event: %s", evt.Event)
	}
}
