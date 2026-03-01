package whatsapp

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New(context.Background(), filepath.Join(t.TempDir(), "wa.db"), "")
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
