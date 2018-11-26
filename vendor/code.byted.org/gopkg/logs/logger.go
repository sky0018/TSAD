package logs

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"path/filepath"

	"code.byted.org/gopkg/env"
)

/*
	toutiao log format:

	{Level} {Data} {Time} {Version}({NumHeaders}) {Location} {HostIP} {PSM} {LogID} {Cluster} {Stage} {RawLog} ...
	Warn 2017-11-28 14:55:22,562 v1(6) kite.go:94 10.8.44.18 toutiao.user.info - default canary message(97)=KITE: processing request error=RPC timeout: context deadline exceeded, remoteIP=10.8.59.153:58834
*/

var (
	spaceBytes   = []byte(" ")
	versionBytes = []byte("v1(6)")
	hostIPBytes  = []byte(env.HostIP())
	psmBytes     = []byte(env.PSM())
	clusterBytes = []byte(env.Cluster())
	stageBytes   = []byte(env.Stage())
	unknownBytes = []byte("-")
)

func init() {
	if len(hostIPBytes) == 0 {
		hostIPBytes = unknownBytes
	}
	if len(psmBytes) == 0 {
		psmBytes = unknownBytes
	}
	if len(clusterBytes) == 0 {
		clusterBytes = unknownBytes
	}
	if len(stageBytes) == 0 {
		stageBytes = unknownBytes
	}
}

func logIDBytes(ctx context.Context) []byte {
	if ctx == nil {
		return unknownBytes
	}

	val := ctx.Value("K_LOGID")
	if val != nil {
		logid := val.(string)
		return []byte(logid)
	}
	return unknownBytes
}

const (
	LevelTrace = iota
	LevelDebug
	LevelInfo
	LevelNotice
	LevelWarn
	LevelError
	LevelFatal
)

var (
	levelMap = []string{
		"Trace",
		"Debug",
		"Info",
		"Notice",
		"Warn",
		"Error",
		"Fatal",
	}

	levelBytes = [][]byte{
		[]byte("Trace"),
		[]byte("Debug"),
		[]byte("Info"),
		[]byte("Notice"),
		[]byte("Warn"),
		[]byte("Error"),
		[]byte("Fatal"),
	}
)

// LogMsg .
type LogMsg struct {
	msg   string
	level int
}

type KVLogMsg struct {
	msg string
	level int
	headers map[string]string
	kvs map[string]string
}

// Logger .
type Logger struct {
	callDepth int // callDepth <= 0 will not print file number info.

	isRunning int32
	level     int
	buf       chan *LogMsg
	flush     chan *sync.WaitGroup
	providers []LogProvider
	
	kvbuf chan *KVLogMsg
	kvproviders []KVLogProvider

	wg   sync.WaitGroup
	stop chan struct{}
}

// NewLogger make default level is debug, default callDepth is 2, default provider is console.
func NewLogger(bufLen int) *Logger {
	l := &Logger{
		level:     LevelDebug,
		buf:       make(chan *LogMsg, bufLen),
		kvbuf:     make(chan *KVLogMsg, bufLen),
		stop:      make(chan struct{}),
		flush:     make(chan *sync.WaitGroup),
		callDepth: 2,
		providers: nil,
	}

	return l
}

// NewConsoleLogger 日志输出到屏幕，通常用于Debug模式
func NewConsoleLogger() *Logger {
	logger := NewLogger(1024)
	consoleProvider := NewConsoleProvider()
	consoleProvider.Init()
	logger.AddProvider(consoleProvider)
	return logger
}

// AddProvider .
func (l *Logger) AddProvider(p LogProvider) error {
	if err := p.Init(); err != nil {
		return err
	}

	if kv, ok := p.(KVLogProvider); ok {
		l.kvproviders = append(l.kvproviders, kv)
	} else {
		l.providers = append(l.providers, p)
	}

	return nil
}

// SetLevel .
func (l *Logger) SetLevel(level int) {
	l.level = level
}

// DisableCallDepth will not print file numbers.
func (l *Logger) DisableCallDepth() {
	l.callDepth = 0
}

// SetCallDepth .
func (l *Logger) SetCallDepth(depth int) {
	l.callDepth = depth
}

// StartLogger .
func (l *Logger) StartLogger() {
	if !atomic.CompareAndSwapInt32(&l.isRunning, 0, 1) {
		return
	}
	if len(l.providers) == 0 && len(l.kvproviders) == 0 {
		fmt.Fprintln(os.Stderr, "logger's providers is nil.")
		return
	}
	l.wg.Add(1)
	go func() {
		defer func() {
			atomic.StoreInt32(&l.isRunning, 0)

			l.cleanBuf()
			for _, provider := range l.providers {
				provider.Flush()
				provider.Destroy()
			}

			l.wg.Done()
		}()
		for {
			select {
			case logMsg, ok := <-l.buf:
				if !ok {
					fmt.Fprintln(os.Stderr, "buf channel has been closed.")
					return
				}
				for _, provider := range l.providers {
					provider.WriteMsg(logMsg.msg, logMsg.level)
				}
			case msg, ok := <-l.kvbuf:
				if !ok {
					fmt.Fprintln(os.Stderr, "kvbuf channel has been closed.")
					return
				}
				for _, provider := range l.kvproviders {
					provider.WriteMsgKVs(msg.level, msg.msg, msg.headers, msg.kvs)
				}
			case wg := <-l.flush:
				l.cleanBuf()
				for _, provider := range l.providers {
					provider.Flush()
				}
				wg.Done()
			case <-l.stop:
				return
			}
		}
	}()
}

func (l *Logger) cleanBuf() {
	for {
		select {
		case msg := <-l.buf:
			for _, provider := range l.providers {
				provider.WriteMsg(msg.msg, msg.level)
			}
		case msg := <-l.kvbuf:
			for _, provider := range l.kvproviders {
				provider.WriteMsgKVs(msg.level, msg.msg, msg.headers, msg.kvs)
			}
		default:
			return
		}
	}
}

// Stop .
func (l *Logger) Stop() {
	if !atomic.CompareAndSwapInt32(&l.isRunning, 1, 0) {
		return
	}
	close(l.stop)
	l.wg.Wait()
}

// Fatal .
func (l *Logger) Fatal(format string, v ...interface{}) {
	if LevelFatal < l.level {
		return
	}
	l.fmtLog(nil, LevelFatal, fmt.Sprintf(format, v...))
}

// CtxFatal .
func (l *Logger) CtxFatal(ctx context.Context, format string, v ...interface{}) {
	if LevelFatal < l.level {
		return
	}
	l.fmtLog(ctx, LevelFatal, fmt.Sprintf(format, v...))
}

// CtxFatalKVs .
func (l *Logger) CtxFatalKVs(ctx context.Context, kvs ...interface{}) {
	if LevelFatal < l.level {
		return
	}
	l.fmtLog(ctx, LevelFatal, "", kvs...)
}

// Error .
func (l *Logger) Error(format string, v ...interface{}) {
	if LevelError < l.level {
		return
	}
	l.fmtLog(nil, LevelError, fmt.Sprintf(format, v...))
}

// CtxError .
func (l *Logger) CtxError(ctx context.Context, format string, v ...interface{}) {
	if LevelError < l.level {
		return
	}
	l.fmtLog(ctx, LevelError, fmt.Sprintf(format, v...))
}

// CtxErrorKVs .
func (l *Logger) CtxErrorKVs(ctx context.Context, kvs ...interface{}) {
	if LevelError < l.level {
		return
	}
	l.fmtLog(ctx, LevelError, "", kvs...)
}

// Warn .
func (l *Logger) Warn(format string, v ...interface{}) {
	if LevelWarn < l.level {
		return
	}
	l.fmtLog(nil, LevelWarn, fmt.Sprintf(format, v...))
}

// CtxWarn .
func (l *Logger) CtxWarn(ctx context.Context, format string, v ...interface{}) {
	if LevelWarn < l.level {
		return
	}
	l.fmtLog(ctx, LevelWarn, fmt.Sprintf(format, v...))
}

// CtxWarnKVs .
func (l *Logger) CtxWarnKVs(ctx context.Context, kvs ...interface{}) {
	if LevelWarn < l.level {
		return
	}
	l.fmtLog(ctx, LevelWarn, "", kvs...)
}

// Notice .
func (l *Logger) Notice(format string, v ...interface{}) {
	if LevelNotice < l.level {
		return
	}
	l.fmtLog(nil, LevelNotice, fmt.Sprintf(format, v...))
}

// CtxNotice .
func (l *Logger) CtxNotice(ctx context.Context, format string, v ...interface{}) {
	if LevelNotice < l.level {
		return
	}
	l.fmtLog(ctx, LevelNotice, fmt.Sprintf(format, v...))
}

// CtxNoticeKVs .
func (l *Logger) CtxNoticeKVs(ctx context.Context, kvs ...interface{}) {
	if LevelNotice < l.level {
		return
	}
	l.fmtLog(ctx, LevelNotice, "", kvs...)
}

// Info .
func (l *Logger) Info(format string, v ...interface{}) {
	if LevelInfo < l.level {
		return
	}
	l.fmtLog(nil, LevelInfo, fmt.Sprintf(format, v...))
}

// CtxInfo .
func (l *Logger) CtxInfo(ctx context.Context, format string, v ...interface{}) {
	if LevelInfo < l.level {
		return
	}
	l.fmtLog(ctx, LevelInfo, fmt.Sprintf(format, v...))
}

// CtxInfoKVs .
func (l *Logger) CtxInfoKVs(ctx context.Context, kvs ...interface{}) {
	if LevelInfo < l.level {
		return
	}
	l.fmtLog(ctx, LevelInfo, "", kvs...)
}

// Debug .
func (l *Logger) Debug(format string, v ...interface{}) {
	if LevelDebug < l.level {
		return
	}
	l.fmtLog(nil, LevelDebug, fmt.Sprintf(format, v...))
}

// CtxDebug .
func (l *Logger) CtxDebug(ctx context.Context, format string, v ...interface{}) {
	if LevelDebug < l.level {
		return
	}
	l.fmtLog(ctx, LevelDebug, fmt.Sprintf(format, v...))
}

// CtxDebugKVs .
func (l *Logger) CtxDebugKVs(ctx context.Context, kvs ...interface{}) {
	if LevelDebug < l.level {
		return
	}
	l.fmtLog(ctx, LevelDebug, "", kvs...)
}

// Trace .
func (l *Logger) Trace(format string, v ...interface{}) {
	if LevelTrace < l.level {
		return
	}
	l.fmtLog(nil, LevelTrace, fmt.Sprintf(format, v...))
}

// CtxTrace .
func (l *Logger) CtxTrace(ctx context.Context, format string, v ...interface{}) {
	if LevelTrace < l.level {
		return
	}
	l.fmtLog(ctx, LevelTrace, fmt.Sprintf(format, v...))
}

// CtxTraceKVs .
func (l *Logger) CtxTraceKVs(ctx context.Context, kvs ...interface{}) {
	if LevelTrace < l.level {
		return
	}
	l.fmtLog(ctx, LevelTrace, "", kvs...)
}

// Warn 2017-11-28 14:55:22,562 v1(6) kite.go:94 10.8.44.18 toutiao.user.info - default canary message(97)=KITE: processing request error=RPC timeout: context deadline exceeded, remoteIP=10.8.59.153:58834
// {Version}({NumHeaders}) {Level} {Data} {Time} {Location} {HostIP} {PSM} {LogID} {Cluster} {Stage} {KV1} {KV2} ...
func (l *Logger) prefixV1(ctx context.Context, level int, writer io.Writer) {
	writer.Write(levelBytes[level])
	writer.Write(spaceBytes)
	dt := timeDate(time.Now())
	writer.Write(dt[:])
	writer.Write(spaceBytes)
	writer.Write(versionBytes)
	writer.Write(spaceBytes)
	writer.Write([]byte(location(l.callDepth + 3)))
	writer.Write(spaceBytes)
	writer.Write(hostIPBytes)
	writer.Write(spaceBytes)
	writer.Write(psmBytes)
	writer.Write(spaceBytes)
	writer.Write(logIDBytes(ctx))
	writer.Write(spaceBytes)
	writer.Write(clusterBytes)
	writer.Write(spaceBytes)
	writer.Write(stageBytes)
	writer.Write(spaceBytes)
}

func (l *Logger) fmtLog(ctx context.Context, level int, rawLog string, kvs ...interface{}) {
	if level < l.level {
		return
	}
	if atomic.LoadInt32(&l.isRunning) == 0 {
		return
	}
	doMetrics(level)

	kvList := getAllKVs(ctx)
	if kvList != nil {
		kvList = append(kvList, kvs...)
		if len(l.providers) > 0 {
			l.fmtForProvider(ctx, level, rawLog, kvList...)
		}
		if len(l.kvproviders) > 0 {
			l.fmtForKVProvider(ctx, level, rawLog, kvList...)
		}
		return
	}

	if len(l.providers) > 0 {
		l.fmtForProvider(ctx, level, rawLog, kvs...)
	}
	if len(l.kvproviders) > 0 {
		l.fmtForKVProvider(ctx, level, rawLog, kvs...)
	}
}

func (l *Logger) fmtForProvider(ctx context.Context, level int, rawLog string, kvs ...interface{}) {
	enc := logEncoderPool.Get().(KVEncoder)
	defer func() {
		enc.Reset()
		logEncoderPool.Put(enc)
	}()

	l.prefixV1(ctx, level, enc)
	enc.AppendKVs(kvs...)

	if len(rawLog) > 0 {
		rawBytes := []byte(rawLog)
		if rawBytes[len(rawBytes)-1] == '\n' {
			rawBytes = rawBytes[:len(rawBytes)-1]
		}
		enc.Write(rawBytes)
	}
	enc.EndRecord()
	msg := enc.String()
	select {
	case l.buf <- &LogMsg{msg: msg, level: level}:
	default:
	}
}

var logEncoderPool = sync.Pool{
	New: func() interface{} {
		return NewTTLogKVEncoder()
	},
}

// Flush 将buf中的日志数据一次性写入到各个provider中，期间新的写入到buf的日志会被丢失
func (l *Logger) Flush() {
	if atomic.LoadInt32(&l.isRunning) == 0 {
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	select {
	case l.flush <- wg:
		wg.Wait()
	case <-time.After(time.Second):
		return // busy ?
	}
}

func (l *Logger) fmtForKVProvider(ctx context.Context, level int, rawLog string, kvs ...interface{}) { 
	headers := make(map[string]string, 9)
	headers["level"] = string(levelBytes[level])
	headers["timestamp"] = strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond)))
	headers["location"] = location(l.callDepth + 2)
	headers["host"] = env.HostIP()
	headers["psm"] = env.PSM()
	headers["cluster"] = env.Cluster()
	headers["logid"] = string(logIDBytes(ctx))
	headers["stage"] = env.Stage()
	headers["pod_name"] = env.PodName()

	kvMap := make(map[string]string, len(kvs))
	for i := 0; i+1 < len(kvs); i += 2 {
		k := kvs[i]
		v := kvs[i+1]
		kstr := string(value2Bytes(k))
		vstr := string(value2Bytes(v))
		kvMap[kstr] = vstr
	}

	msg := &KVLogMsg{
		msg: rawLog,
		level: level,
		headers: headers,
		kvs: kvMap,
	}
	select {
	case l.kvbuf <- msg:
	default:
		// TODO(zhangyuanjia): do metrics?
	}
}

func location(deep int) string {
	_, file, line, ok := runtime.Caller(deep)
	if !ok {
		file = "???"
		line = 0
	}
	return filepath.Base(file) + ":" + strconv.Itoa(line)
}
