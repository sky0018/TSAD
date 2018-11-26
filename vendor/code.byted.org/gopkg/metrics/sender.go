package metrics

import (
	"bytes"
	"io"
	"net"
	"sort"
	"strings"
	"time"
)

const (
	// DO NOT MODIFY IT IF YOU DONT KNOWN WHAT YOU ARE DOING
	maxBunchBytes = 32 << 10 // 32kb

	// send metrics immediately if larger than the size
	maxPendingSize = 4096

	// 200ms timeout before send metrics
	emitInterval = 200 * time.Millisecond
)

const (
	_emit = "emit"
)

type singleMetric struct {
	mt     string
	prefix string
	name   string
	v      float64

	tbuf []byte
	tags []T // for cache use only
}

func (m *singleMetric) SetTags(tags []T, ext []byte) {
	SortTags(tags)
	n := 0
	for _, v := range tags {
		n += len(v.Name) + 1 + len(v.Value)
	}
	if len(tags) > 0 { // "|"
		n += len(tags) - 1
		if len(ext) > 0 {
			n += 1 // "|"
		}
	}
	n += len(ext)
	b := make([]byte, n)
	p := b[:0]
	for i, t := range tags {
		if i != 0 {
			p = append(p, '|')
		}
		p = append(p, t.Name...)
		p = append(p, '=')
		p = append(p, t.Value...)
	}
	if len(tags) > 0 && len(ext) > 0 {
		p = append(p, '|')
	}
	p = append(p, ext...)
	if len(p) != len(b) {
		panic("bufsize err")
	}
	m.tbuf = b
}

func (m singleMetric) Less(o singleMetric) bool {
	if c := strings.Compare(m.prefix, o.prefix); c != 0 {
		return c < 0
	}
	if c := strings.Compare(m.name, o.name); c != 0 {
		return c < 0
	}
	return bytes.Compare(m.tbuf, o.tbuf) < 0
}

func (m singleMetric) Equal(o singleMetric) bool {
	if m.prefix == o.prefix &&
		m.name == o.name &&
		bytes.Equal(m.tbuf, o.tbuf) {
		return true
	}
	return false
}

func (m *singleMetric) MarshalSize() int {
	if m.tbuf == nil {
		panic("metric buf == nil")
	}
	// protocol: 6 fields: emit $type $prefix.name  $value $tag ""
	n := 0
	n += msgpackArrayHeaderSize
	n += msgpackStringSize(_emit)
	n += msgpackStringSize(m.mt)
	if len(m.prefix) > 0 {
		n += msgpackStringHeaderSize + (len(m.prefix) + 1 + len(m.name))
	} else {
		n += msgpackStringSize(m.name)
	}
	n += msgpackStringHeaderSize + floatStrSize(m.v) // int64 + "." + 5 prec float + str header
	n += msgpackStringHeaderSize + len(m.tbuf)
	n += msgpackStringHeaderSize + 0
	return n
}

func (m *singleMetric) MarshalTo(b []byte) {
	if m.tbuf == nil {
		panic("metric buf == nil")
	}
	p := b[:0]
	// protocol: 6 fields: emit $type $prefix.name  $value $tag ""
	p = msgpackAppendArrayHeader(p, 6)
	p = msgpackAppendString(p, _emit)
	p = msgpackAppendString(p, m.mt)
	if len(m.prefix) > 0 {
		p = msgpackAppendStringHeader(p, uint16(len(m.prefix)+1+len(m.name)))
		p = append(p, m.prefix...)
		p = append(p, '.')
		p = append(p, m.name...)
	} else {
		p = msgpackAppendString(p, m.name)
	}
	p = msgpackAppendStringHeader(p, uint16(floatStrSize(m.v)))
	p = appendFloat64(p, m.v)
	p = msgpackAppendStringHeader(p, uint16(len(m.tbuf)))
	p = append(p, m.tbuf...)
	p = msgpackAppendString(p, "")
	if len(p) != len(b) {
		panic("buf size err")
	}
}

type metricsWriter struct {
	addr string
	err  error
	conn net.Conn

	ctime time.Time
}

func (w *metricsWriter) Write(b []byte) (n int, err error) {
	n = len(b)
	if w.conn == nil {
		if w.addr == BlackholeAddr {
			return
		}
		now := time.Now()
		if now.Sub(w.ctime) < time.Second {
			return
		}
		w.ctime = now
		if strings.HasPrefix(w.addr, "/") {
			w.conn, err = net.Dial("unixgram", w.addr)
		} else {
			w.conn, err = net.Dial("udp", w.addr)
		}
		if err != nil {
			println("metrics conn err: ", err.Error())
			w.conn = nil
			return
		}
	}
	_, err = w.conn.Write(b)
	if err != nil {
		w.conn = nil
	}
	return
}

func hasUnixSocket(path string) bool {
	for i := 0; i < 3; i++ {
		_, err := net.Dial("unixgram", path)
		if err == nil {
			return true
		}
	}
	return false
}

type sender struct {
	addr      string
	makeBunch bool

	counterCh chan singleMetric
	storeCh   chan singleMetric
	timerCh   chan singleMetric
}

func newSender(addr string) *sender {
	s := &sender{addr: addr}
	if addr == DefaultMetricsServer && hasUnixSocket(DefaultMetricsSock) {
		s.addr = DefaultMetricsSock
		s.makeBunch = true
	}
	s.counterCh = make(chan singleMetric, 4096*4)
	s.storeCh = make(chan singleMetric, 4096*4)
	s.timerCh = make(chan singleMetric, 4096*4)

	s.runLoops()
	return s
}

func (s *sender) SendCounter(m singleMetric) error {
	select {
	case s.counterCh <- m:
	default:
		return ErrEmitTimeout
	}
	return nil
}

func (s *sender) SendStore(m singleMetric) error {
	select {
	case s.storeCh <- m:
	default:
		return ErrEmitTimeout
	}
	return nil
}

func (s *sender) SendTimer(m singleMetric) error {
	select {
	case s.timerCh <- m:
	default:
		return ErrEmitTimeout
	}
	return nil
}

func (s *sender) runLoops() {
	go s.counterLoop()
	go s.storeLoop()
	go s.timerLoop()
}

func (s *sender) counterLoop() {
	metrics := make([]singleMetric, 0, maxPendingSize)
	t := time.Tick(emitInterval)
	w := &metricsWriter{addr: s.addr}
	for {
		select {
		case <-t:
			if len(metrics) > 0 {
				s.sendcounter(w, metrics)
				metrics = metrics[:0]
			}
		case m := <-s.counterCh:
			metrics = append(metrics, m)
			if len(metrics) > maxPendingSize {
				s.sendcounter(w, metrics)
				metrics = metrics[:0]
			}
		}
	}
}

func (s *sender) storeLoop() {
	metrics := make([]singleMetric, 0, maxPendingSize)
	t := time.Tick(emitInterval)
	w := &metricsWriter{addr: s.addr}
	for {
		select {
		case <-t:
			if len(metrics) > 0 {
				s.send(w, metrics)
				metrics = metrics[:0]
			}
		case m := <-s.storeCh:
			metrics = append(metrics, m)
			if len(metrics) > maxPendingSize {
				s.send(w, metrics)
				metrics = metrics[:0]
			}
		}
	}
}

func (s *sender) timerLoop() {
	metrics := make([]singleMetric, 0, maxPendingSize)
	t := time.Tick(emitInterval)
	w := &metricsWriter{addr: s.addr}
	for {
		select {
		case <-t:
			if len(metrics) > 0 {
				s.send(w, metrics)
				metrics = metrics[:0]
			}
		case m := <-s.timerCh:
			metrics = append(metrics, m)
			if len(metrics) >= maxPendingSize {
				s.send(w, metrics)
				metrics = metrics[:0]
			}
		}
	}
}

type byMetricKey []singleMetric

func (k byMetricKey) Len() int      { return len(k) }
func (k byMetricKey) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
func (k byMetricKey) Less(i, j int) bool {
	return k[i].Less(k[j])
}

func (s *sender) sendcounter(w io.Writer, ms []singleMetric) {
	if len(ms) == 0 {
		return
	}
	sort.Sort(byMetricKey(ms))
	p := ms[:1]
	for i := range ms[1:] {
		if p[len(p)-1].Equal(ms[i]) {
			p[len(p)-1].v += ms[i].v
		} else {
			p = append(p, ms[i])
		}
	}
	s.send(w, p)
}

func (s *sender) sendbunch(w io.Writer, ms []singleMetric) int {
	// cal size
	k := 0
	n := msgpackArrayHeaderSize
	for _, m := range ms {
		n += m.MarshalSize()
		k++
		if n >= maxBunchBytes {
			break
		}
	}
	ms = ms[:k]
	// marshal to bb
	bb := make([]byte, n)
	i := len(msgpackAppendArrayHeader(bb[:0], uint16(len(ms))))
	for _, m := range ms {
		size := m.MarshalSize()
		m.MarshalTo(bb[i : i+size])
		i += size
	}

	if i != len(bb) {
		panic("write buf err")
	}
	w.Write(bb)
	return k
}

func (s *sender) send(w io.Writer, ms []singleMetric) {
	if !s.makeBunch {
		b := make([]byte, 0, 64<<10)
		for _, m := range ms {
			n := m.MarshalSize()
			if n > cap(b) {
				b = make([]byte, 0, 2*cap(b))
			}
			b = b[:n]
			m.MarshalTo(b)
			w.Write(b)
		}
		return
	}
	// send bunch
	for len(ms) > 0 {
		k := s.sendbunch(w, ms)
		ms = ms[k:]
	}
	if len(ms) > 0 {
		s.sendbunch(w, ms)
	}
}
