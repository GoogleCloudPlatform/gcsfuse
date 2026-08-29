// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelslog provides [Handler], an [slog.Handler] implementation, that
// can be used to bridge between the [log/slog] API and [OpenTelemetry].
//
// # Record Conversion
//
// The [slog.Record] are converted to OpenTelemetry [log.Record] in the following
// way:
//
//   - Time is set as the Timestamp.
//   - Message is set as the Body using a [log.StringValue].
//   - Level is transformed and set as the Severity. The SeverityText is also
//     set.
//   - PC is dropped.
//   - Attr are transformed and set as the Attributes.
//
// The Level is transformed by using the static offset to the OpenTelemetry
// Severity types. For example:
//
//   - [slog.LevelDebug] is transformed to [log.SeverityDebug]
//   - [slog.LevelInfo] is transformed to [log.SeverityInfo]
//   - [slog.LevelWarn] is transformed to [log.SeverityWarn]
//   - [slog.LevelError] is transformed to [log.SeverityError]
//
// Attribute values are transformed based on their [slog.Kind]:
//
//   - [slog.KindAny] values are transformed based on their type or
//     into a string value encoded using [fmt.Sprintf] if there is no matching type.
//   - [slog.KindBool] are transformed to [log.BoolValue] directly.
//   - [slog.KindDuration] are transformed to [log.Int64Value] as nanoseconds.
//   - [slog.KindFloat64] are transformed to [log.Float64Value] directly.
//   - [slog.KindInt64] are transformed to [log.Int64Value] directly.
//   - [slog.KindString] are transformed to [log.StringValue] directly.
//   - [slog.KindTime] are transformed to [log.Int64Value] as nanoseconds since
//     the Unix epoch.
//   - [slog.KindUint64] are transformed to [log.Int64Value] using int64
//     conversion.
//   - [slog.KindGroup] are transformed to [log.MapValue] using appropriate
//     transforms for each group value.
//   - [slog.KindLogValuer] the value is resolved and then transformed.
//
// [OpenTelemetry]: https://opentelemetry.io/docs/concepts/signals/logs/
package otelslog // import "go.opentelemetry.io/contrib/bridges/otelslog"

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// NewLogger returns a new [slog.Logger] backed by a new [Handler]. See
// [NewHandler] for details on how the backing Handler is created.
func NewLogger(name string, options ...Option) *slog.Logger {
	return slog.New(NewHandler(name, options...))
}

type config struct {
	provider   log.LoggerProvider
	version    string
	schemaURL  string
	attributes []attribute.KeyValue
	source     bool
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	return c
}

func (c config) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	if c.attributes != nil {
		opts = append(opts, log.WithInstrumentationAttributes(c.attributes...))
	}
	return c.provider.Logger(name, opts...)
}

// Option configures a [Handler].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [Handler]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [Handler]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithAttributes returns an [Option] that configures the instrumentation scope
// attributes of the [log.Logger] used by a [Handler].
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optFunc(func(c config) config {
		c.attributes = attributes
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [Handler] to create its [log.Logger].
//
// By default if this Option is not provided, the Handler will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// WithSource returns an [Option] that configures the [Handler] to include
// the source location of the log record in log attributes.
func WithSource(source bool) Option {
	return optFunc(func(c config) config {
		c.source = source
		return c
	})
}

// Handler is an [slog.Handler] that sends all logging records it receives to
// OpenTelemetry. See package documentation for how conversions are made.
type Handler struct {
	// Ensure forward compatibility by explicitly making this not comparable.
	noCmp [0]func() //nolint:unused  // This is indeed used.

	attrs  *kvBuffer
	group  *group
	logger log.Logger

	source bool
}

// Compile-time check *Handler implements slog.Handler.
var _ slog.Handler = (*Handler)(nil)

// NewHandler returns a new [Handler] to be used as an [slog.Handler].
//
// If [WithLoggerProvider] is not provided, the returned Handler will use the
// global LoggerProvider.
//
// The provided name needs to uniquely identify the code being logged. This is
// most commonly the package name of the code. If name is empty, the
// [log.Logger] implementation may override this value with a default.
func NewHandler(name string, options ...Option) *Handler {
	cfg := newConfig(options)
	return &Handler{
		logger: cfg.logger(name),
		source: cfg.source,
	}
}

// Handle handles the passed record.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	h.logger.Emit(ctx, h.convertRecord(record))
	return nil
}

func (h *Handler) convertRecord(r slog.Record) log.Record {
	var record log.Record
	record.SetTimestamp(r.Time)
	record.SetBody(log.StringValue(r.Message))

	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	record.SetSeverity(log.Severity(r.Level + sevOffset))
	record.SetSeverityText(r.Level.String())

	if h.source {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		record.AddAttributes(
			log.String(string(semconv.CodeFilePathKey), f.File),
			log.String(string(semconv.CodeFunctionNameKey), f.Function),
			log.Int(string(semconv.CodeLineNumberKey), f.Line),
		)
	}

	if h.attrs != nil && h.attrs.err != nil {
		record.SetErr(h.attrs.err)
	}
	if h.attrs.Len() > 0 {
		record.AddAttributes(h.attrs.KeyValues()...)
	}

	n := r.NumAttrs()
	if h.group != nil {
		if err := h.group.Err(); err != nil {
			record.SetErr(err)
		}

		if n > 0 {
			buf := newKVBuffer(n)
			r.Attrs(buf.AddAttr)
			if buf.err != nil {
				record.SetErr(buf.err)
			}
			if buf.Len() > 0 {
				record.AddAttributes(h.group.KeyValue(buf.KeyValues()...))
			} else {
				// A Handler should not output groups if there are no attributes.
				g := h.group.NextNonEmpty()
				if g != nil {
					record.AddAttributes(g.KeyValue())
				}
			}
		} else {
			// A Handler should not output groups if there are no attributes.
			g := h.group.NextNonEmpty()
			if g != nil {
				record.AddAttributes(g.KeyValue())
			}
		}
	} else if n > 0 {
		buf := newKVBuffer(n)
		r.Attrs(buf.AddAttr)
		if buf.err != nil {
			record.SetErr(buf.err)
		}
		record.AddAttributes(buf.KeyValues()...)
	}

	return record
}

// Enabled returns true if the Handler is enabled to log for the provided
// context and Level. Otherwise, false is returned if it is not enabled.
func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	param := log.EnabledParameters{Severity: log.Severity(l + sevOffset)}
	return h.logger.Enabled(ctx, param)
}

// WithAttrs returns a new [slog.Handler] based on h that will log using the
// passed attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	if h2.group != nil {
		h2.group = h2.group.Clone()
		h2.group.AddAttrs(attrs)
	} else {
		if h2.attrs == nil {
			h2.attrs = newKVBuffer(len(attrs))
		} else {
			h2.attrs = h2.attrs.Clone()
		}
		h2.attrs.AddAttrs(attrs)
	}
	return &h2
}

// WithGroup returns a new [slog.Handler] based on h that will log all messages
// and attributes within a group of the provided name.
func (h *Handler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.group = &group{name: name, next: h2.group}
	return &h2
}

// group represents a group received from slog.
type group struct {
	// name is the name of the group.
	name string
	// attrs are the attributes associated with the group.
	attrs *kvBuffer
	// next points to the next group that holds this group.
	//
	// Groups are represented as map value types in OpenTelemetry. This means
	// that for an slog group hierarchy like the following ...
	//
	//   WithGroup("G").WithGroup("H").WithGroup("I")
	//
	// the corresponding OpenTelemetry log value types will have the following
	// hierarchy ...
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{
	//       Key: "H",
	//       Value: []KeyValue{{
	//         Key: "I",
	//         Value: []KeyValue{},
	//       }},
	//     }},
	//   }
	//
	// When attributes are recorded (i.e. Info("msg", "key", "value") or
	// WithAttrs("key", "value")) they need to be added to the "leaf" group. In
	// the above example, that would be group "I":
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{
	//       Key: "H",
	//       Value: []KeyValue{{
	//         Key: "I",
	//         Value: []KeyValue{
	//           String("key", "value"),
	//         },
	//       }},
	//     }},
	//   }
	//
	// Therefore, groups are structured as a linked-list with the "leaf" node
	// being the head of the list. Following the above example, the group data
	// representation would be ...
	//
	//   *group{"I", next: *group{"H", next: *group{"G"}}}
	next *group
}

// Err returns the error to use from g's linked-list (including g itself). If
// no error is found, nil is returned.
func (g *group) Err() error {
	for g != nil {
		if g.attrs != nil && g.attrs.err != nil {
			return g.attrs.err
		}
		g = g.next
	}
	return nil
}

// NextNonEmpty returns the next group within g's linked-list that has
// attributes (including g itself). If no group is found, nil is returned.
func (g *group) NextNonEmpty() *group {
	if g == nil || g.attrs.Len() > 0 {
		return g
	}
	return g.next.NextNonEmpty()
}

// KeyValue returns group g containing kvs as a [log.KeyValue]. The value of
// the returned KeyValue will be of type [log.KindMap].
//
// The passed kvs are rendered in the returned value, but are not added to the
// group.
//
// This does not check g. It is the callers responsibility to ensure g is
// non-empty or kvs is non-empty so as to return a valid group representation
// (according to slog).
func (g *group) KeyValue(kvs ...log.KeyValue) log.KeyValue {
	// Assumes checking of group g already performed (i.e. non-empty).
	out := log.Map(g.name, g.attrs.KeyValues(kvs...)...)
	g = g.next
	for g != nil {
		// A Handler should not output groups if there are no attributes.
		if g.attrs.Len() > 0 {
			out = log.Map(g.name, g.attrs.KeyValues(out)...)
		}
		g = g.next
	}
	return out
}

// Clone returns a copy of g.
func (g *group) Clone() *group {
	if g == nil {
		return nil
	}
	g2 := *g
	g2.attrs = g2.attrs.Clone()
	return &g2
}

// AddAttrs add attrs to g.
func (g *group) AddAttrs(attrs []slog.Attr) {
	if g.attrs == nil {
		g.attrs = newKVBuffer(len(attrs))
	}
	g.attrs.AddAttrs(attrs)
}

type kvBuffer struct {
	data []log.KeyValue
	err  error
}

func newKVBuffer(n int) *kvBuffer {
	return &kvBuffer{data: make([]log.KeyValue, 0, n)}
}

// Len returns the number of [log.KeyValue] held by b.
func (b *kvBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.data)
}

// Clone returns a copy of b.
func (b *kvBuffer) Clone() *kvBuffer {
	if b == nil {
		return nil
	}
	return &kvBuffer{data: slices.Clone(b.data), err: b.err}
}

// KeyValues returns kvs appended to the [log.KeyValue] held by b.
func (b *kvBuffer) KeyValues(kvs ...log.KeyValue) []log.KeyValue {
	if b == nil {
		return kvs
	}
	return append(b.data, kvs...)
}

// AddAttrs adds attrs to b.
func (b *kvBuffer) AddAttrs(attrs []slog.Attr) {
	b.data = slices.Grow(b.data, len(attrs))
	for _, a := range attrs {
		_ = b.AddAttr(a)
	}
}

// AddAttr adds attr to b and returns true.
//
// This is designed to be passed to the AddAttributes method of an
// [slog.Record].
//
// If attr is a group with an empty key, its values will be flattened.
//
// If attr is empty, it will be dropped.
func (b *kvBuffer) AddAttr(attr slog.Attr) bool {
	if attr.Value.Kind() == slog.KindAny {
		if err, ok := attr.Value.Any().(error); ok {
			b.err = err
			return true
		}
	}

	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			// A Handler should inline the Attrs of a group with an empty key.
			for _, a := range attr.Value.Group() {
				_ = b.AddAttr(a)
			}
			return true
		}

		if attr.Value.Any() == nil {
			// A Handler should ignore an empty Attr.
			return true
		}
	}
	b.data = append(b.data, log.KeyValue{
		Key:   attr.Key,
		Value: convert(attr.Value),
	})
	return true
}

func convert(v slog.Value) log.Value {
	switch v.Kind() {
	case slog.KindAny:
		return convertValue(v.Any())
	case slog.KindBool:
		return log.BoolValue(v.Bool())
	case slog.KindDuration:
		return log.Int64Value(v.Duration().Nanoseconds())
	case slog.KindFloat64:
		return log.Float64Value(v.Float64())
	case slog.KindInt64:
		return log.Int64Value(v.Int64())
	case slog.KindString:
		return log.StringValue(v.String())
	case slog.KindTime:
		return log.Int64Value(v.Time().UnixNano())
	case slog.KindUint64:
		const maxInt64 = ^uint64(0) >> 1
		u := v.Uint64()
		if u > maxInt64 {
			return log.Float64Value(float64(u))
		}
		return log.Int64Value(int64(u))
	case slog.KindGroup:
		g := v.Group()
		buf := newKVBuffer(len(g))
		buf.AddAttrs(g)
		return log.MapValue(buf.data...)
	case slog.KindLogValuer:
		return convert(v.Resolve())
	default:
		// Try to handle this as gracefully as possible.
		//
		// Don't panic here. The goal here is to have developers find this
		// first if a new slog.Kind is added. A test on the new kind will find
		// this malformed attribute as well as a panic. However, it is
		// preferable to have user's open issue asking why their attributes
		// have a "unhandled: " prefix than say that their code is panicking.
		return log.StringValue(fmt.Sprintf("unhandled: (%s) %+v", v.Kind(), v.Any()))
	}
}
