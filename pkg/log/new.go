package log

import "context"

type Logger interface {
	Debug(ctx context.Context, arg ...any)
	Debugf(ctx context.Context, templete string, arg ...any)
	Info(ctx context.Context, arg ...any)
	Infof(ctx context.Context, templete string, arg ...any)
	Warn(ctx context.Context, arg ...any)
	Warnf(ctx context.Context, templete string, arg ...any)
	Error(ctx context.Context, arg ...any)
	Errorf(ctx context.Context, templete string, arg ...any)
	Fatal(ctx context.Context, arg ...any)
	Fatalf(ctx context.Context, templete string, arg ...any)
}

type nopLogger struct{}

func (l *nopLogger) Debug(ctx context.Context, arg ...any)                 {}
func (l *nopLogger) Debugf(ctx context.Context, templete string, arg ...any) {}
func (l *nopLogger) Info(ctx context.Context, arg ...any)                  {}
func (l *nopLogger) Infof(ctx context.Context, templete string, arg ...any)  {}
func (l *nopLogger) Warn(ctx context.Context, arg ...any)                  {}
func (l *nopLogger) Warnf(ctx context.Context, templete string, arg ...any)  {}
func (l *nopLogger) Error(ctx context.Context, arg ...any)                 {}
func (l *nopLogger) Errorf(ctx context.Context, templete string, arg ...any) {}
func (l *nopLogger) Fatal(ctx context.Context, arg ...any)                 {}
func (l *nopLogger) Fatalf(ctx context.Context, templete string, arg ...any) {}

func NewNopLogger() Logger {
	return &nopLogger{}
}