package logs

import (
	"context"
	"sync"
)

const (
	noticeCtxKey = "K_NOTICE"
)

func getNotice(ctx context.Context) *noticeKVs {
	i := ctx.Value(noticeCtxKey)
	if ntc, ok := i.(*noticeKVs); ok {
		return ntc
	}
	return nil
}

type noticeKVs struct {
	kvs []interface{}
	sync.Mutex
}

func newNoticeKVs() *noticeKVs {
	return &noticeKVs{
		kvs: make([]interface{}, 0, 16),
	}
}

func (l *noticeKVs) PushNotice(k, v interface{}) {
	l.Lock()
	l.kvs = append(l.kvs, k, v)
	l.Unlock()
}

func (l *noticeKVs) KVs() []interface{} {
	l.Lock()
	kvs := l.kvs
	l.Unlock()
	return kvs
}
