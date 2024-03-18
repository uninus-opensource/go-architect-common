package log

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/uninus-opensource/uninus-go-architect-common/uuid"

	logkit "github.com/go-kit/kit/log"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	//LogTime is log key for timestamp
	LogTime = "ts"
	//LogCaller is log key for source file name
	LogCaller = "caller"
	//LogMethod is log key for method name
	LogMethod = "method"
	//LogUser is log key for user
	LogUser = "user"
	//LogEmail is log key for email
	LogEmail = "email"
	//LogMobile is log key for mobile no
	LogMobile = "mobile"
	//LogRole is log key for role
	LogRole = "role"
	//LogTook is log key for call duration
	LogTook = "took"
	//LogInfo is log key for info
	LogInfo = "[INFO]"
	//LogDebug is log key for debug
	LogDebug = "[DEBUG]"
	//LogCritical is log key for critical
	LogCritical = "[CRITICAL]"
	//LogError is log key for error
	LogError = "[ERROR]"
	//LogBasic is log key for basic log
	LogBasic = "[BASIC]"
	//LogWarning is log key for warning log
	LogWarning = "[WARNING]"
	//LogReq is log key for request log
	LogReq = "[REQUEST]"
	//LogResp is log key for response log
	LogResp = "[RESPONSE]"
	//LogData is log key for data log
	LogData = "[DATA]"
	//LogService is log key for service name
	LogService = "service"
	//LogToken is log key for token
	LogToken = "token"
	//LogExit is log key for exit
	LogExit = "exit"
	//default file logger
	logFile = "service.log"
)

type ConfigLog struct {
	Caller int
}

// File set default log to file
func File(file string) {
	logFile := &lumberjack.Logger{
		Filename:  file,
		MaxSize:   1, // megabytes
		LocalTime: true,
		Compress:  true, // disabled by default
	}

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags)
}

// Logger returns default logger
func Logger() logkit.Logger {
	File(logFile)
	logger := logkit.NewLogfmtLogger(NewDefaultLogWriter())
	logger = logkit.With(logger, LogCaller, logkit.DefaultCaller)

	return logger
}

// StdLogger returns logger to stderr
func StdLogger() logkit.Logger {
	logger := logkit.NewLogfmtLogger(os.Stderr)
	logger = logkit.With(logger, LogTime, logkit.DefaultTimestampUTC, LogCaller, logkit.DefaultCaller)

	return logger
}

// StdLoggerConf returns logger to stderr with config
func StdLoggerConf(conf ConfigLog) logkit.Logger {
	logger := logkit.NewLogfmtLogger(os.Stderr)
	logger = logkit.With(logger, LogTime, logkit.DefaultTimestampUTC, LogCaller, logkit.Caller(conf.Caller))

	return logger
}

// FileLogger returns file logger
func FileLogger(file string) logkit.Logger {
	File(file)
	logger := logkit.NewLogfmtLogger(NewDefaultLogWriter())
	logger = logkit.With(logger, LogCaller, logkit.DefaultCaller)

	return logger
}

func StackDriverLogger() logkit.Logger {
	logger := NewSDLogger(os.Stdout)
	logger = logkit.With(logger, LogTime, logkit.DefaultTimestampUTC, LogCaller, logkit.DefaultCaller)

	return logger
}

// ConsoleLog is console log format in service. this for helping logging in service
type ConsoleLog struct {
	TraceID   string
	RequestID string
	SpecialID map[string]interface{}
	Log       string
	TimeStart time.Time
	UserID    string
}

// GenerateConsoleLog is genereate log
func (cslog *ConsoleLog) GenerateConsoleLog(ctx context.Context) {
	var logs []string
	cslog.TraceID = GetTraceID(ctx)
	cslog.RequestID = GetRequestID(ctx)
	logs = append(logs, fmt.Sprintf("trace_id = '%s'", cslog.TraceID))
	logs = append(logs, fmt.Sprintf("request_id = '%s'", cslog.RequestID))
	if len(cslog.SpecialID) > 0 {
		for key, val := range cslog.SpecialID {
			logs = append(logs, fmt.Sprintf("%s = '%v'", key, val))
		}
	}
	if cslog.UserID != "" {
		logs = append(logs, fmt.Sprintf("user_id = '%s'", cslog.UserID))
	}
	cslog.Log = strings.Join(logs, " , ")
	cslog.TimeStart = time.Now()
}

// GetTimeSince is get duration service running.
func (cslog *ConsoleLog) GetTimeSince() float64 {
	return time.Since(cslog.TimeStart).Seconds()
}

func GetTraceID(ctx context.Context) string {
	if md, ok := runtime.ServerMetadataFromContext(ctx); ok {
		if val, exists := md.HeaderMD["trace.id"]; exists {
			return val[0]
		} else {
			traceID, _ := uuid.New()
			trid := traceID.String()
			md.HeaderMD.Append("trace.id", trid)
			return trid
		}
	}

	traceID, _ := uuid.New()
	trid := traceID.String()
	return trid
}

func GetRequestID(ctx context.Context) string {
	if md, ok := runtime.ServerMetadataFromContext(ctx); ok {
		if val, exists := md.HeaderMD["transaction.id"]; exists {
			return val[0]
		}
	}
	txID, _ := uuid.New()
	return txID.String()
}

func GetTraceIDFromHTTPContext(req *http.Request) string {
	if val := req.Header.Get("trace.id"); val != "" {
		return val
	}

	traceID, _ := uuid.New()
	trid := traceID.String()
	req.Header.Set("trace.id", trid)
	return trid
}

func GetTrxIDFromHTTPContext(req *http.Request) string {
	if val := req.Header.Get("transaction.id"); val != "" {
		return val
	}

	txID, _ := uuid.New()
	return txID.String()
}

func ConvertStatus(status int32) []byte {
	return []byte(fmt.Sprintf("%d", status))
}
