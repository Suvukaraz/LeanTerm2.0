package services

import (
	"context"
	"sync"

	"github.com/Suvukaraz/PureTerm/internal/events"
	"github.com/Suvukaraz/PureTerm/internal/session"
	"github.com/google/uuid"
)

// Emitter sends events to the GUI.
type Emitter interface {
	Emit(event string, data any)
}

// DialogBroker bridges backend questions (passwords, host keys, ...) to the UI.
// The backend calls Prompt, which emits "dialog:request" and blocks until the
// GUI answers through DialogService.Respond (or the context ends).
// It is not a service of its own, so Prompt is not exposed to the GUI.
type DialogBroker struct {
	mu      sync.Mutex
	pending map[string]chan session.PromptResponse
	emitter Emitter
}

func NewDialogBroker(emitter Emitter) *DialogBroker {
	return &DialogBroker{
		pending: make(map[string]chan session.PromptResponse),
		emitter: emitter,
	}
}

// DialogService is the GUI-facing part of the broker.
type DialogService struct {
	broker *DialogBroker
}

func NewDialogService(broker *DialogBroker) *DialogService {
	return &DialogService{broker: broker}
}

// Respond delivers the user's answer for a pending dialog request.
func (ds *DialogService) Respond(requestID string, resp session.PromptResponse) {
	if ds.broker != nil {
		ds.broker.respond(requestID, resp)
	}
}

// Prompt implements session.Prompter.
func (ds *DialogBroker) Prompt(ctx context.Context, req session.PromptRequest) (session.PromptResponse, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	ch := make(chan session.PromptResponse, 1)

	ds.mu.Lock()
	ds.pending[req.ID] = ch
	emitter := ds.emitter
	ds.mu.Unlock()

	defer func() {
		ds.mu.Lock()
		delete(ds.pending, req.ID)
		ds.mu.Unlock()
	}()

	if emitter == nil {
		return session.PromptResponse{Cancelled: true}, session.ErrPromptCancelled
	}
	emitter.Emit(events.DialogRequest, req)

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		emitter.Emit(events.DialogCancel, map[string]any{"id": req.ID})
		return session.PromptResponse{Cancelled: true}, ctx.Err()
	}
}

func (ds *DialogBroker) respond(requestID string, resp session.PromptResponse) {
	ds.mu.Lock()
	ch, ok := ds.pending[requestID]
	ds.mu.Unlock()
	if ok {
		select {
		case ch <- resp:
		default:
		}
	}
}
