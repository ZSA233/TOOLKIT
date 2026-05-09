package app

import (
	"fmt"
	"strings"
)

type taskProgressLogEmitter struct {
	prefix    string
	emit      func(string)
	lastLabel string
	lastDone  int
	lastTotal int
}

func newTaskProgressLogEmitter(prefix string, emit func(string)) *taskProgressLogEmitter {
	return &taskProgressLogEmitter{
		prefix: prefix,
		emit:   emit,
	}
}

func (emitter *taskProgressLogEmitter) Log(done int, total int, label string) {
	if emitter == nil || emitter.emit == nil {
		return
	}
	if total < 1 {
		total = 1
	}
	normalizedLabel := strings.TrimSpace(label)
	if normalizedLabel == "" {
		normalizedLabel = "working"
	}
	// Progress callbacks can repeat the same state during startup and cancellation.
	// Collapse identical updates so the task log stays readable during long sweeps.
	if normalizedLabel == emitter.lastLabel && done == emitter.lastDone && total == emitter.lastTotal {
		return
	}
	emitter.lastLabel = normalizedLabel
	emitter.lastDone = done
	emitter.lastTotal = total
	emitter.emit(fmt.Sprintf("%s %d/%d: %s", emitter.prefix, done, total, normalizedLabel))
}
