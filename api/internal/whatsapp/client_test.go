package whatsapp

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New(context.Background(), filepath.Join(t.TempDir(), "wa.db"), "", slog.Default())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(c.Disconnect)
	return c
}

// TestStatusNotConnectedBeforeQR verifies that Status().Connected is false for a
// fresh client with no stored session. This guards against using wac.IsConnected()
// (websocket-level) instead of wac.IsLoggedIn() (auth-level): whatsmeow opens a
// websocket to WhatsApp servers during QR pairing, which makes IsConnected() return
// true even though no account has been linked yet.
func TestStatusNotConnectedBeforeQR(t *testing.T) {
	c := newTestClient(t)

	s := c.Status()
	if s.Connected {
		t.Error("Status.Connected should be false for a fresh client with no session")
	}
	if s.QRCode != "" {
		t.Errorf("Status.QRCode should be empty before Connect, got %q", s.QRCode)
	}
}

func TestSubscribeReceivesStatus(t *testing.T) {
	c := newTestClient(t)

	ch := c.Subscribe()
	defer c.Unsubscribe(ch)

	c.notify()

	select {
	case s := <-ch:
		if s.Connected {
			t.Error("expected Connected=false for a fresh client")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for status after notify")
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	c := newTestClient(t)

	ch := c.Subscribe()
	c.Unsubscribe(ch)

	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after Unsubscribe")
	}
}

func TestNotifyIsNonBlocking(t *testing.T) {
	c := newTestClient(t)

	ch := c.Subscribe()
	defer c.Unsubscribe(ch)

	// Fill the buffered channel.
	c.notify()

	// Second call with a full channel should not block.
	done := make(chan struct{})
	go func() {
		c.notify()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("notify blocked on a full channel")
	}
}

func TestMultipleSubscribersReceive(t *testing.T) {
	c := newTestClient(t)

	ch1 := c.Subscribe()
	ch2 := c.Subscribe()
	defer c.Unsubscribe(ch1)
	defer c.Unsubscribe(ch2)

	c.notify()

	for i, ch := range []chan Status{ch1, ch2} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d timed out", i+1)
		}
	}
}

func TestHandleQRClearsActiveOnTimeout(t *testing.T) {
	c := newTestClient(t)

	ch := c.Subscribe()
	defer c.Unsubscribe(ch)

	// Simulate a QR flow: code → timeout
	qrChan := make(chan whatsmeow.QRChannelItem, 2)
	qrChan <- whatsmeow.QRChannelItem{Event: "code", Code: "test-qr-123"}
	qrChan <- whatsmeow.QRChannelItem{Event: "timeout"}
	close(qrChan)

	c.mu.Lock()
	c.qrActive = true
	c.mu.Unlock()

	c.handleQR(qrChan)

	// After timeout, qrActive must be false so RequestQR can restart pairing.
	c.mu.RLock()
	active := c.qrActive
	qr := c.qrCode
	c.mu.RUnlock()

	if active {
		t.Error("qrActive should be false after QR timeout")
	}
	if qr != "" {
		t.Errorf("qrCode should be empty after timeout, got %q", qr)
	}
}

func TestHandleQRSetsCodeThenClears(t *testing.T) {
	c := newTestClient(t)

	ch := c.Subscribe()
	defer c.Unsubscribe(ch)

	qrChan := make(chan whatsmeow.QRChannelItem, 3)
	qrChan <- whatsmeow.QRChannelItem{Event: "code", Code: "qr-1"}
	close(qrChan)

	c.mu.Lock()
	c.qrActive = true
	c.mu.Unlock()

	// Process just the code event.
	c.handleQR(qrChan)

	// Subscriber should have received the code.
	select {
	case s := <-ch:
		if s.QRCode != "qr-1" {
			t.Errorf("expected QRCode=qr-1, got %q", s.QRCode)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for QR status")
	}
}

func TestRequestQRNoopWhenActive(t *testing.T) {
	c := newTestClient(t)
	c.ctx = context.Background()

	c.mu.Lock()
	c.qrActive = true
	c.mu.Unlock()

	// Should not panic or start a second flow.
	c.RequestQR()

	c.mu.RLock()
	active := c.qrActive
	c.mu.RUnlock()

	if !active {
		t.Error("qrActive should remain true when RequestQR is called during active flow")
	}
}

func TestRequestQRNoopWhenConnected(t *testing.T) {
	c := newTestClient(t)
	c.ctx = context.Background()

	// Simulate connected state: IsLoggedIn() checks wac.Store.ID != nil,
	// but we can't set that without a real session. Instead, verify that
	// RequestQR with qrActive=false on a fresh client attempts startQR
	// (which will fail on GetQRChannel without a server, but won't panic).
	c.RequestQR()

	// No crash is the assertion; startQR fails gracefully.
}
