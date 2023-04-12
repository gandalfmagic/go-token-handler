package zlogger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gandalfmagic/realip"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

type Logger struct {
	*zap.Logger
	lastErr            error
	lastErrDescription string
	traceID            string
	spanID             string
}

// NewLogger creates a new Logger configured with the specified parameters.
//
// The level parameter configure the default logging level, the parameter is a string
// and the possible values are:
//   - `debug`:  all the existing logs, usually disabled in  production
//   - `info`:   the default logging level
//   - `warn`:   more important than Info, but don't need individual human review
//   - `error`:  high-priority errors, if an application is running smoothly,
//     it shouldn't generate any error-level logs
//   - `dpanic`: particularly important errors, in development the logger panics
//     after writing the message
//   - `panic:   logs a message, then panics
//   - `fatal`:  logs a message, then calls os.Exit(1)
//
// the production parameter configures the logger for the development environment, if
// the value is `false`, or for the production environment if the value is `true`
func NewLogger(level string, production bool) (*Logger, error) {
	var encoderConfig zapcore.EncoderConfig
	if production {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = ""

	var encoder zapcore.Encoder
	if production {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var l zapcore.Level
	if err := l.Set(level); err != nil {
		return nil, err
	}

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), zap.NewAtomicLevelAt(l))

	var log *zap.Logger
	if production {
		log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel))
	} else {
		log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel), zap.Development())
	}

	return &Logger{Logger: log}, nil
}

func NewContext(parent context.Context, z *Logger) context.Context {
	return context.WithValue(parent, ctxKey{}, z)
}

func FromContext(ctx context.Context) *Logger {
	l, _ := ctx.Value(ctxKey{}).(*Logger)

	if traceID := trace.SpanFromContext(ctx).SpanContext().TraceID(); traceID.IsValid() {
		l.traceID = traceID.String()
	} else {
		l.traceID = ""
	}

	if spanID := trace.SpanFromContext(ctx).SpanContext().SpanID(); spanID.IsValid() {
		l.spanID = spanID.String()
	} else {
		l.spanID = ""
	}

	return l
}

// Sync synchronize the logger output by flushing any buffered log entries.
// Applications should take care to call Sync before exiting.
func (l *Logger) Sync() {
	l.Debug("shutting down the logger")
	_ = l.Logger.Sync()
}

// SetError saves the error value and an associated message into the Logger state.
// These values will then be used in the Middleware function to add context to
// the HTTP error.
//
// SetError should be called every time is important to display the original error
// text inside a specific log.
//
// The data will be used by the Middleware function to add context to the logger
// output. The `description` value will be used as message for the log entry, and
// add the `err` value will be added as a field to the log body.
// After the log entry is written, the state of the Logger will be cleaned.
//
// Note: if you also need to return an error to the REST client, you should use
// the JsonError function.
func (l *Logger) SetError(description string, err error) {
	l.lastErrDescription = description
	l.lastErr = err
}

// JsonError generate the JSON data to send to the client in case of an error.
//
// JsonError should be called every time you need to return an error to the
// REST client of your application.
//
// The output will be formatted using the following json template:
//
//		{
//	      "code": status code,
//	      "status": "textual representation of the status code",
//	      "message" "description",
//	      "timestamp": "now()",
//		}
//
// The message will only be populated when the description parameter is not an
// empty value.
//
// The data will also  be used by the Middleware function to add context to the
// logger output. The `description` value will be used as message for the log
// entry, and add the `err` value will be added as a field to the log body.
// After the log entry is written, the state of the Logger will be cleaned.
//
// Note: if you need to add context to the logger, but without returning an
// error to the REST client, you should use the SetError function.
func (l *Logger) JsonError(w http.ResponseWriter, code int, description string, err error) {
	l.lastErrDescription = description
	l.lastErr = err

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	data := struct {
		Code      int       `json:"code"`
		Status    string    `json:"status"`
		Message   string    `json:"message,omitempty"`
		Timestamp time.Time `json:"timestamp"`
		err       error
	}{
		Code:      code,
		Status:    http.StatusText(code),
		Message:   description,
		Timestamp: time.Now(),
		err:       err,
	}

	_ = json.NewEncoder(w).Encode(data)
}

// Middleware is used to wrap other HTTP handlers, to log all the requests with
// the meaningful details needed for an HTTP server
func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// clean the state of the Logger
			l.lastErrDescription = ""
			l.lastErr = nil
		}()

		// override the ResponseWriter to record the status code and the
		// body size of the response
		recorder := &StatusRecorder{ResponseWriter: w, Status: 200}
		w = recorder

		ctx := context.WithValue(r.Context(), ctxKey{}, l)

		// call the wrapped handler and measure the time taken to run it
		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))
		latency := time.Since(start).String()

		// prepare the standard log fields
		fields := []zapcore.Field{
			zap.Int("status", recorder.Status),
			zap.Int("size", recorder.Written),
			zap.String("latency", latency),
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.String("host", r.RemoteAddr),
		}

		if l.traceID != "" {
			fields = append(fields, zap.String("trace-id", l.traceID))
		}

		if l.spanID != "" {
			fields = append(fields, zap.String("trace-id", l.spanID))
		}

		// add the `referer` field only if the Header exists in the request
		if r.Header.Get("Referer") != "" {
			fields = append(fields, zap.String("referer", r.Header.Get("Referer")))
		}

		// get the real ip address of the request and add it as the `remote_ip` field
		if realIP, err := realip.Get(r); err == nil {
			fields = append(fields, zap.String("remote_ip", realIP))
		}

		// add the `protocol` and the `user_agent` fields from the request
		fields = append(fields,
			zap.String("protocol", r.Proto),
			zap.String("user_agent", r.UserAgent()),
		)

		// if the `lastErr` value is set in the Logger state, is added as a field
		if l.lastErr != nil {
			fields = append(fields, zap.Error(l.lastErr))
		}

		// if the `lastErrDescription` value is set in the logger state, it's used
		// as detail for the log message
		var message string
		if l.lastErrDescription != "" {
			message = fmt.Sprintf("%s: %s", http.StatusText(recorder.Status), l.lastErrDescription)
		} else {
			message = http.StatusText(recorder.Status)
		}

		// the Status code is used to define the logging level of the entry
		switch {
		case recorder.Status >= 500:
			l.Error(message, fields...)
		case recorder.Status >= 400:
			l.Warn(message, fields...)
		case recorder.Status >= 300:
			l.Info(message, fields...)
		default:
			l.Info(message, fields...)
		}
	})
}
