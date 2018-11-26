package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type HourClock struct {
	stop chan struct{}
}

func NewHourTicker() <-chan time.Time {
	hourClock := &HourClock{stop: make(chan struct{})}
	return hourClock.C()
}

func (hc *HourClock) C() <-chan time.Time {
	ch := make(chan time.Time)
	go func() {
		hour := time.Now().Hour()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case t := <-ticker.C:
				if t.Hour() != hour {
					ch <- t
					hour = t.Hour()
				}
			case <-hc.stop:
				return
			}
		}
	}()
	return ch
}

func (hc *HourClock) Stop() {
	hc.stop <- struct{}{}
}

type SegDuration string

const (
	HourDur SegDuration = "Hour"
	DayDur  SegDuration = "Day"
	NoDur   SegDuration = "No"
)

type FileProvider struct {
	sync.Mutex
	enableRotate bool
	hourTicker   <-chan time.Time

	fd       *os.File
	filename string
	level    int
}

// NOTE(xiangchao): 由于基于size大小的切割方案可能会导致切割出来的文件在命名上存在一些问题，因此这里废弃基于size大小的切割方案
func NewFileProvider(filename string, dur SegDuration, size int64) *FileProvider {
	rotate := false
	if dur != NoDur {
		rotate = true
	}

	provider := &FileProvider{
		enableRotate: rotate,
		filename:     filename,
		level:        LevelDebug,
		hourTicker:   NewHourTicker(),
	}

	return provider
}

func (fp *FileProvider) Init() error {
	var (
		fd  *os.File
		err error
	)
	realFile, err := fp.timeFilename()
	if err != nil {
		return err
	}
	if env := os.Getenv("IS_PROD_RUNTIME"); len(env) == 0 {
		fd, err = os.OpenFile(realFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	} else {
		fd, err = os.OpenFile(realFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	}
	fp.fd = fd
	_, err = os.Lstat(fp.filename)
	if err == nil || os.IsExist(err) {
		os.Remove(fp.filename)
	}
	os.Symlink("./"+filepath.Base(realFile), fp.filename)
	return nil
}

func (fp *FileProvider) doCheck() error {
	fp.Lock()
	defer fp.Unlock()

	if !fp.enableRotate {
		return nil
	}

	select {
	case <-fp.hourTicker:
		if err := fp.truncate(); err != nil {
			fmt.Fprintf(os.Stderr, "truncate file %s error: %s\n", fp.filename, err)
			return err
		}
	default:
	}
	return nil
}

func (fp *FileProvider) SetLevel(l int) {
	fp.level = l
}

func (fp *FileProvider) WriteMsg(msg string, level int) error {
	if level < fp.level {
		return nil
	}
	// NOTE(xiangchao): 按照size切割已经被忽略
	fp.doCheck()
	_, err := fmt.Fprint(fp.fd, msg)
	return err
}

func (fp *FileProvider) Destroy() error {
	return fp.fd.Close()
}

func (fp *FileProvider) Flush() error {
	return fp.fd.Sync()
}

// 1: 拼接出新的日志文件的名字
// 2: 拷贝当前日志文件到新的文件
// 3: Truncate当前日志文件
func (fp *FileProvider) truncate() error {
	fp.fd.Sync()
	fp.fd.Close()
	return fp.Init()
}

func (fp *FileProvider) timeFilename() (string, error) {
	absPath, err := filepath.Abs(fp.filename)
	if err != nil {
		return "", err
	}
	return absPath + "." + time.Now().Format("2006-01-02_15"), nil
}
