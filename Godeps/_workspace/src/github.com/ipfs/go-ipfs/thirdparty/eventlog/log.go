package eventlog

import (
	"fmt"
	"time"

	context "github.com/ipfs/go-ipfs/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/ipfs/go-ipfs/util" // TODO remove IPFS dependency
)

// StandardLogger provides API compatibility with standard printf loggers
// eg. go-logging
type StandardLogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

// EventLogger extends the StandardLogger interface to allow for log items
// containing structured metadata
type EventLogger interface {
	StandardLogger

	// Event merges structured data from the provided inputs into a single
	// machine-readable log event.
	//
	// If the context contains metadata, a copy of this is used as the base
	// metadata accumulator.
	//
	// If one or more loggable objects are provided, these are deep-merged into base blob.
	//
	// Next, the event name is added to the blob under the key "event". If
	// the key "event" already exists, it will be over-written.
	//
	// Finally the timestamp and package name are added to the accumulator and
	// the metadata is logged.
	Event(ctx context.Context, event string, m ...Loggable)

	EventBegin(ctx context.Context, event string, m ...Loggable) *EventInProgress
}

// Logger retrieves an event logger by name
func Logger(system string) EventLogger {

	// TODO if we would like to adjust log levels at run-time. Store this event
	// logger in a map (just like the util.Logger impl)
	return &eventLogger{system: system, StandardLogger: util.Logger(system)}
}

// eventLogger implements the EventLogger and wraps a go-logging Logger
type eventLogger struct {
	StandardLogger

	system string
	// TODO add log-level
}

func (el *eventLogger) EventBegin(ctx context.Context, event string, metadata ...Loggable) *EventInProgress {
	start := time.Now()
	el.Event(ctx, fmt.Sprintf("%sBegin", event), metadata...)

	eip := &EventInProgress{}
	eip.doneFunc = func(additional []Loggable) {

		metadata = append(metadata, additional...)                      // anything added during the operation
		metadata = append(metadata, LoggableMap(map[string]interface{}{ // finally, duration of event
			"duration": time.Now().Sub(start),
		}))

		el.Event(ctx, event, metadata...)
	}
	return eip
}

func (el *eventLogger) Event(ctx context.Context, event string, metadata ...Loggable) {

	// short circuit if theres nothing to write to
	if !WriterGroup.Active() {
		return
	}

	// Collect loggables for later logging
	var loggables []Loggable

	// get any existing metadata from the context
	existing, err := MetadataFromContext(ctx)
	if err != nil {
		existing = Metadata{}
	}
	loggables = append(loggables, existing)

	for _, datum := range metadata {
		loggables = append(loggables, datum)
	}

	e := entry{
		loggables: loggables,
		system:    el.system,
		event:     event,
	}

	e.Log() // TODO replace this when leveled-logs have been implemented
}

type EventInProgress struct {
	loggables []Loggable
	doneFunc  func([]Loggable)
}

// Append adds loggables to be included in the call to Done
func (eip *EventInProgress) Append(l Loggable) {
	eip.loggables = append(eip.loggables, l)
}

// SetError includes the provided error
func (eip *EventInProgress) SetError(err error) {
	eip.loggables = append(eip.loggables, LoggableMap{
		"error": err.Error(),
	})
}

// Done creates a new Event entry that includes the duration and appended
// loggables.
func (eip *EventInProgress) Done() {
	eip.doneFunc(eip.loggables) // create final event with extra data
}

// Close is an alias for done
func (eip *EventInProgress) Close() error {
	eip.Done()
	return nil
}
