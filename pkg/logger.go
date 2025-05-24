package pkg

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var Log *LoggerService

type LoggerService struct {
	log zerolog.Logger
	env string
}

func NewLoggerService(env string, file *os.File) *LoggerService {
	var output io.Writer

	if env == "development" {
		// Logging to both file and std.out during development
		fileOut := zerolog.ConsoleWriter{
			Out:        file,
			TimeFormat: time.RFC3339,
			NoColor:    true,
		}
		consoleOut := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		output = zerolog.MultiLevelWriter(consoleOut, fileOut)

	} else if env == "production" {
		// Logging only to file during production
		output = zerolog.ConsoleWriter{Out: file, TimeFormat: time.RFC3339}

	} else {
		panic(errors.New("Could not identify environment"))
	}

	logger := zerolog.New(output).With().Timestamp().Logger()
	return &LoggerService{
		log: logger,
		env: env,
	}
}

// Service setup loggers (not for API use)
func (l *LoggerService) SetupInfo(msg string) {
	l.log.WithLevel(zerolog.InfoLevel).Msg(msg)
}
func (l *LoggerService) SetupFail(msg string, err error) {
	l.log.WithLevel(zerolog.ErrorLevel).Err(err).Msg(msg)
}

// Loggers for API use
func (l *LoggerService) Info(c *gin.Context, msg string) {
	l.log.WithLevel(zerolog.InfoLevel).
		Str("req_id", GrabRequestId(c)).
		Str("path", c.FullPath()).
		Str("method", c.Request.Method).
		Msgf("%s", msg)
}

func (l *LoggerService) Debug(c *gin.Context, msg string) {
	l.log.WithLevel(zerolog.DebugLevel).
		Str("req_id", GrabRequestId(c)).
		Str("path", c.FullPath()).
		Str("method", c.Request.Method).
		Msgf("%s", msg)
}

func (l *LoggerService) Warn(c *gin.Context, msg string) {
	l.log.WithLevel(zerolog.WarnLevel).
		Str("req_id", GrabRequestId(c)).
		Str("path", c.FullPath()).
		Str("method", c.Request.Method).
		Msgf("%s", msg)
}

func (l *LoggerService) Error(c *gin.Context, msg string, err error) {
	l.log.WithLevel(zerolog.ErrorLevel).
		Str("req_id", GrabRequestId(c)).
		Str("path", c.FullPath()).
		Str("method", c.Request.Method).
		Err(err).
		Msgf("%s", msg)
}

func (l *LoggerService) Fatal(c *gin.Context, msg string, err error) {
	l.log.WithLevel(zerolog.FatalLevel).
		Str("req_id", GrabRequestId(c)).
		Err(err).
		Msgf("%s", msg)
}

func (l *LoggerService) Success(c *gin.Context) {
	l.log.WithLevel(zerolog.InfoLevel).
		Str("req_id", GrabRequestId(c)).
		Str("path", c.FullPath()).
		Str("method", c.Request.Method).
		Msgf("[SUCCESS]: Webhook processed successfully.")
}
