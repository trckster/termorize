package db

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type wordEventsKey struct{}
type wordEvents struct{ ids []uuid.UUID }

var wordEventSink struct {
	sync.RWMutex
	handler func(uuid.UUID) bool
}

func SetWordCreatedHandler(handler func(uuid.UUID) bool) func() {
	wordEventSink.Lock()
	previous := wordEventSink.handler
	wordEventSink.handler = handler
	wordEventSink.Unlock()
	return func() {
		wordEventSink.Lock()
		wordEventSink.handler = previous
		wordEventSink.Unlock()
	}
}

// WordTransaction defers events to the outer commit and discards events from rolled-back savepoints.
// Every transaction enclosing word creation must use this wrapper.
func WordTransaction(conn *gorm.DB, fn func(*gorm.DB) error) error {
	events, nested := conn.Statement.Context.Value(wordEventsKey{}).(*wordEvents)
	if !nested {
		if _, inTransaction := conn.Statement.ConnPool.(gorm.TxCommitter); inTransaction {
			return errors.New("word creation requires an outer WordTransaction")
		}
		events = &wordEvents{}
		conn = conn.WithContext(context.WithValue(conn.Statement.Context, wordEventsKey{}, events))
	}
	start := len(events.ids)
	committed := false
	defer func() {
		if !committed {
			events.ids = events.ids[:start]
		}
	}()
	if err := conn.Transaction(fn); err != nil {
		return err
	}
	committed = true
	if !nested {
		wordEventSink.RLock()
		handler := wordEventSink.handler
		wordEventSink.RUnlock()
		if handler != nil {
			for _, id := range events.ids {
				handler(id)
			}
		}
	}
	return nil
}

func RecordWordCreated(conn *gorm.DB, id uuid.UUID) error {
	events, ok := conn.Statement.Context.Value(wordEventsKey{}).(*wordEvents)
	if !ok {
		return errors.New("word creation requires WordTransaction")
	}
	// Larger transactions can fall back to the hourly NULL sweep without unbounded memory.
	if len(events.ids) < 2048 {
		events.ids = append(events.ids, id)
	}
	return nil
}
