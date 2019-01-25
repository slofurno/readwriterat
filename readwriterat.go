package readwriterat

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

const DefaultDownloadPartSize = 1024 * 1024 * 5
const DefaultDownloadConcurrency = 5

type WriterAtReader struct {
	free          chan *bytes.Buffer
	toread        chan *bytes.Buffer
	inuse         map[int]*bytes.Buffer
	mu            sync.Mutex
	nextRead      int
	currentReader *bytes.Buffer

	currentBuffers int
	lastRead       int

	PartSize    int64
	Concurrency int
	Debug       bool
}

func New(opts ...func(*WriterAtReader)) *WriterAtReader {
	wat := &WriterAtReader{
		inuse:       map[int]*bytes.Buffer{},
		PartSize:    DefaultDownloadPartSize,
		Concurrency: DefaultDownloadConcurrency,
	}

	for i := range opts {
		opts[i](wat)
	}

	wat.free = make(chan *bytes.Buffer, wat.Concurrency*4)
	wat.toread = make(chan *bytes.Buffer, wat.Concurrency*2)
	return wat
}

func (w *WriterAtReader) Close() {
	w.checkReadable(true)
	close(w.toread)
}

func (w *WriterAtReader) Read(p []byte) (int, error) {
	if w.currentReader == nil {
		if w.Debug {
			fmt.Println("getting next reader", w.lastRead)
		}
		w.lastRead = 1
		w.currentReader = <-w.toread
	}

	if w.currentReader.Len() == 0 {
		w.free <- w.currentReader
		var ok bool
		if w.Debug {
			fmt.Println("getting next reader", w.lastRead)
			w.lastRead++
		}
		w.currentReader, ok = <-w.toread
		if !ok {
			return 0, io.EOF
		}
	}
	return w.currentReader.Read(p)
}

func (w *WriterAtReader) waitFree() *bytes.Buffer {
	select {
	case buf := <-w.free:
		buf.Reset()
		return buf
	}
}
func (w *WriterAtReader) getFree() *bytes.Buffer {
	select {
	case buf := <-w.free:
		buf.Reset()
		return buf
	default:
		w.currentBuffers++
		if w.Debug {
			fmt.Println("making free:", w.currentBuffers)
		}
		return bytes.NewBuffer(make([]byte, 0, w.PartSize))
	}
}

func (w *WriterAtReader) checkReadable(flush bool) {
	for i := w.nextRead; ; i++ {
		buf, ok := w.inuse[i]
		if !ok {
			return
		}

		if flush {
			w.toread <- buf
			continue
		}

		if int64(buf.Len()) < w.PartSize {
			return
		}

		w.nextRead++
		if w.Debug {
			fmt.Println("readable:", i)
		}
		w.toread <- buf
	}
}

func (w *WriterAtReader) WriteAt(p []byte, off int64) (int, error) {
	target := int(off / w.PartSize)

	var buf *bytes.Buffer
	var ok bool

	w.mu.Lock()
	defer w.mu.Unlock()
	buf, ok = w.inuse[target]
	if !ok {
		if w.Debug {
			fmt.Println("writing:", target)
		}
		buf = w.getFree()
		w.inuse[target] = buf
	}

	n, err := buf.Write(p)
	if err != nil {
		return -1, err
	}

	w.checkReadable(false)
	return n, err
}
