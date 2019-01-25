package readwriterat

import (
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

type chunkWriter struct {
	src         []byte
	concurrency int
}

func (w chunkWriter) WriteChunks(writer io.WriterAt) {
	partsize := int(len(w.src) / w.concurrency)
	wg := sync.WaitGroup{}
	wg.Add(w.concurrency)

	for i := 0; i < w.concurrency; i++ {
		go func(i int) {
			if i == 0 {
				time.Sleep(100 * time.Millisecond)
			}
			chunk := w.src[i*partsize : (i+1)*partsize]
			writer.WriteAt(chunk, int64(i*partsize))
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestSlowReader(t *testing.T) {
	src, _ := ioutil.ReadFile("./readwriterat.go")
	concurrency := 20

	cw := chunkWriter{
		src:         src,
		concurrency: concurrency,
	}

	writer := New()
	writer.Debug = true
	writer.PartSize = int64(len(src) / concurrency)

	go func() {
		cw.WriteChunks(writer)
		writer.Close()
	}()
	fmt.Println("next")

	ioutil.ReadAll(writer)
}
