package whatsapp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// EventHandler is called for each incoming WhatsApp event.
type EventHandler func(evt any)

// Client wraps a whatsmeow client with session management and QR pairing.
type Client struct {
	wac       *whatsmeow.Client
	container *sqlstore.Container
	name      string // push name to set on first pairing

	mu     sync.RWMutex
	qrCode string // current QR code string, empty when connected or expired

	subsMu sync.Mutex
	subs   map[chan Status]struct{}
}

// Status represents the current WhatsApp connection state.
type Status struct {
	Connected bool   `json:"connected"`
	QRCode    string `json:"qr,omitempty"`
}

// New creates a whatsmeow client backed by a SQLite session store.
// name is the WhatsApp push name set on first QR pairing (e.g. "Rekan").
// The client is not connected yet; call Connect to start.
func New(ctx context.Context, dbPath, name string) (*Client, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)
	container, err := sqlstore.New(ctx, "sqlite", dsn, nil)
	if err != nil {
		return nil, fmt.Errorf("whatsapp store: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp device: %w", err)
	}

	wac := whatsmeow.NewClient(device, nil)

	c := &Client{
		wac:       wac,
		container: container,
		name:      name,
		subs:      make(map[chan Status]struct{}),
	}

	// Notify subscribers on connect/disconnect events.
	// Use a goroutine to avoid blocking the whatsmeow event loop.
	wac.AddEventHandler(func(evt any) {
		switch evt.(type) {
		case *events.Connected, *events.Disconnected, *events.LoggedOut:
			go c.notify()
		}
	})

	return c, nil
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
		Connected: c.wac.IsLoggedIn(),
		QRCode:    qr,
	}
}

// Subscribe returns a channel that receives a Status push on every state change.
// The caller must call Unsubscribe when done to avoid leaking the channel.
func (c *Client) Subscribe() chan Status {
	ch := make(chan Status, 1)
	c.subsMu.Lock()
	c.subs[ch] = struct{}{}
	c.subsMu.Unlock()
	return ch
}

// Unsubscribe removes ch from the subscriber list and closes it.
func (c *Client) Unsubscribe(ch chan Status) {
	c.subsMu.Lock()
	delete(c.subs, ch)
	c.subsMu.Unlock()
	close(ch)
}

func (c *Client) notify() {
	s := c.Status()
	c.subsMu.Lock()
	for ch := range c.subs {
		select {
		case ch <- s:
		default: // drop if reader is slow
		}
	}
	c.subsMu.Unlock()
}

// AddEventHandler registers a handler for WhatsApp events (messages, receipts, etc.).
func (c *Client) AddEventHandler(handler EventHandler) {
	c.wac.AddEventHandler(whatsmeow.EventHandler(handler))
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

// ResolveLID resolves a LID JID to its phone number JID, or returns the input unchanged
// if it is not a LID or cannot be resolved.
func (c *Client) ResolveLID(ctx context.Context, jid types.JID) types.JID {
	if jid.Server != types.HiddenUserServer {
		return jid
	}
	if c == nil {
		return types.EmptyJID
	}
	pn, err := c.wac.Store.GetAltJID(ctx, jid)
	if err != nil || pn.IsEmpty() {
		return types.EmptyJID
	}
	return pn
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
		c.notify()
		log.Printf("whatsapp qr event: %s", evt.Event)

		if evt == whatsmeow.QRChannelSuccess && c.name != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := c.wac.SendAppState(ctx, appstate.BuildSettingPushName(c.name)); err != nil {
				log.Printf("whatsapp: failed to set push name %q: %v", c.name, err)
			}
			cancel()
		}
	}
}
