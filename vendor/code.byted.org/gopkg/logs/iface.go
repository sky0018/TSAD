package logs

import (
	"code.byted.org/gopkg/context"
)

// logger的接口，所有的logger都应该实现该接口
type LoggerInterface interface {
	Fatal(format string, v ...interface{})
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Notice(format string, v ...interface{})
	Info(format string, v ...interface{})
	Debug(format string, v ...interface{})
	Trace(format string, v ...interface{})
}

// ctxLogger接口
type CtxLoggerInterface interface {
	CtxFatal(ctx context.Context, format string, v ...interface{})
	CtxError(ctx context.Context, format string, v ...interface{})
	CtxWarn(ctx context.Context, format string, v ...interface{})
	CtxNotice(ctx context.Context, format string, v ...interface{})
	CtxInfo(ctx context.Context, format string, v ...interface{})
	CtxDebug(ctx context.Context, format string, v ...interface{})
	CtxTrace(ctx context.Context, format string, v ...interface{})
}

// ctxlogfmt接口
type CtxLogfmtInterface interface {
	CtxFatalKvs(ctx context.Context, kvs ...interface{})
	CtxErrorKvs(ctx context.Context, kvs ...interface{})
	CtxWarnKvs(ctx context.Context, kvs ...interface{})
	CtxNoticeKvs(ctx context.Context, kvs ...interface{})
	CtxInfoKvs(ctx context.Context, kvs ...interface{})
	CtxDebugKvs(ctx context.Context, kvs ...interface{})
	CtxTraceKvs(ctx context.Context, kvs ...interface{})
}
