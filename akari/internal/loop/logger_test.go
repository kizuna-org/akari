package loop

import (
	"context"
	"log/slog"
)

// recordingLogger returns a logger that keeps what it was told, so tests can
// check that what became conscious was put on the record.
func recordingLogger() (*slog.Logger, *[]slog.Record) {
	records := new([]slog.Record)

	return slog.New(&recordingHandler{records: records}), records
}

// recordingHandler collects records in memory.
type recordingHandler struct {
	records *[]slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	*h.records = append(*h.records, record)

	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }
