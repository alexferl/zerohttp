package main

import (
	"context"
	"os"

	"github.com/alexferl/zerohttp/log"
	"github.com/rs/zerolog"
)

// ZerologAdapter wraps zerolog.Logger to implement our Logger interface
type ZerologAdapter struct {
	logger zerolog.Logger
}

// NewZerologAdapter creates a new zerolog adapter
func NewZerologAdapter(logger zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger: logger}
}

// NewZerologAdapterDefault creates a zerolog adapter with default configuration
func NewZerologAdapterDefault() *ZerologAdapter {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &ZerologAdapter{logger: logger}
}

func (z *ZerologAdapter) Debug(msg string, fields ...log.Field) {
	event := z.logger.Debug()
	z.addFields(event, fields...)
	event.Msg(msg)
}

func (z *ZerologAdapter) Info(msg string, fields ...log.Field) {
	event := z.logger.Info()
	z.addFields(event, fields...)
	event.Msg(msg)
}

func (z *ZerologAdapter) Warn(msg string, fields ...log.Field) {
	event := z.logger.Warn()
	z.addFields(event, fields...)
	event.Msg(msg)
}

func (z *ZerologAdapter) Error(msg string, fields ...log.Field) {
	event := z.logger.Error()
	z.addFields(event, fields...)
	event.Msg(msg)
}

func (z *ZerologAdapter) Panic(msg string, fields ...log.Field) {
	event := z.logger.Panic()
	z.addFields(event, fields...)
	event.Msg(msg)
}

func (z *ZerologAdapter) Fatal(msg string, fields ...log.Field) {
	event := z.logger.Fatal()
	z.addFields(event, fields...)
	event.Msg(msg)
}

func (z *ZerologAdapter) WithFields(fields ...log.Field) log.Logger {
	ctx := z.logger.With()
	for _, field := range fields {
		ctx = ctx.Interface(field.Key, field.Value)
	}
	return &ZerologAdapter{logger: ctx.Logger()}
}

func (z *ZerologAdapter) WithContext(ctx context.Context) log.Logger {
	return &ZerologAdapter{logger: z.logger.With().Ctx(ctx).Logger()}
}

func (z *ZerologAdapter) addFields(event *zerolog.Event, fields ...log.Field) {
	for _, field := range fields {
		switch v := field.Value.(type) {
		case error:
			event.Err(v)
		case string:
			event.Str(field.Key, v)
		case int:
			event.Int(field.Key, v)
		case int64:
			event.Int64(field.Key, v)
		case float64:
			event.Float64(field.Key, v)
		case bool:
			event.Bool(field.Key, v)
		default:
			event.Interface(field.Key, v)
		}
	}
}
