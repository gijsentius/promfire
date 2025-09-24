package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

// Logger provides structured JSON logging with configurable levels
type Logger struct {
	level     LogLevel
	component string
}

var globalLogger *Logger

// Init initializes the global logger
func Init(level LogLevel, component string) {
	globalLogger = &Logger{
		level:     level,
		component: component,
	}
}

// SetLevel changes the current log level
func SetLevel(level LogLevel) {
	if globalLogger != nil {
		globalLogger.level = level
	}
}

// GetLevel returns the current log level
func GetLevel() LogLevel {
	if globalLogger != nil {
		return globalLogger.level
	}
	return INFO
}

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "TRACE":
		return TRACE
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// getCaller returns the caller information
func getCaller() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return ""
	}

	// Get just the filename, not the full path
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]

	return fmt.Sprintf("%s:%d", filename, line)
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Component: l.component,
		Fields:    fields,
		Caller:    getCaller(),
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to standard logging if JSON marshal fails
		log.Printf("ERROR: Failed to marshal log entry: %v", err)
		return
	}

	fmt.Println(string(jsonData))

	// Exit on fatal errors
	if level == FATAL {
		os.Exit(1)
	}
}

// Global logging functions
func Trace(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		return
	}
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	globalLogger.log(TRACE, message, f)
}

func Debug(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		return
	}
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	globalLogger.log(DEBUG, message, f)
}

func Info(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		return
	}
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	globalLogger.log(INFO, message, f)
}

func Warn(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		return
	}
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	globalLogger.log(WARN, message, f)
}

func Error(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		return
	}
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	globalLogger.log(ERROR, message, f)
}

func Fatal(message string, fields ...map[string]interface{}) {
	if globalLogger == nil {
		return
	}
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	globalLogger.log(FATAL, message, f)
}

// Convenience functions with formatting
func Infof(format string, args ...interface{}) {
	Info(fmt.Sprintf(format, args...))
}

func Debugf(format string, args ...interface{}) {
	Debug(fmt.Sprintf(format, args...))
}

func Tracef(format string, args ...interface{}) {
	Trace(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...interface{}) {
	Warn(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

func Fatalf(format string, args ...interface{}) {
	Fatal(fmt.Sprintf(format, args...))
}