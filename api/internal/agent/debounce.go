package agent

import (
	"strings"
	"sync"
	"time"
)

const debounceWindow = 2 * time.Second

// Debouncer collects rapid-fire messages per operator and fires once after a quiet period.
type Debouncer struct {
	mu      sync.Mutex
	pending map[string]*pendingOp
}

type pendingOp struct {
	messages []string
	timer    *time.Timer
}

func NewDebouncer() *Debouncer {
	return &Debouncer{
		pending: make(map[string]*pendingOp),
	}
}

// Submit adds a message for the given operator JID. After debounceWindow of silence,
// process is called with all concatenated messages. process is called in a new goroutine.
func (d *Debouncer) Submit(jid, text string, process func(combined string)) {
	d.mu.Lock()
	defer d.mu.Unlock()

	op, exists := d.pending[jid]
	if exists {
		op.timer.Stop()
		op.messages = append(op.messages, text)
	} else {
		op = &pendingOp{
			messages: []string{text},
		}
		d.pending[jid] = op
	}

	op.timer = time.AfterFunc(debounceWindow, func() {
		d.mu.Lock()
		msgs := op.messages
		delete(d.pending, jid)
		d.mu.Unlock()

		process(strings.Join(msgs, " "))
	})
}
