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
	partsize    int64
}

func (w chunkWriter) WriteChunks(writer io.WriterAt) {
	wg := sync.WaitGroup{}
	wg.Add(w.concurrency)

	for i := 0; i < w.concurrency; i++ {
		go func(i int) {
			if i == 0 {
				time.Sleep(100 * time.Millisecond)
			}
			chunk := w.src[i*int(w.partsize) : (i+1)*int(w.partsize)]
			writer.WriteAt(chunk, int64(i*int(w.partsize)))
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestSlowReader(t *testing.T) {
	src, err := ioutil.ReadFile("./readwriterat.go")
	if err != nil {
		t.Fatal(err)
	}
	concurrency := 20
	partsize := int64(len(src) / concurrency)
	expected := src[:int64(concurrency)*partsize]

	cw := chunkWriter{
		src:         src,
		concurrency: concurrency,
		partsize:    partsize,
	}

	writer := New()
	writer.Debug = true
	writer.PartSize = partsize

	go func() {
		cw.WriteChunks(writer)
		writer.Close()
	}()

	actual, err := ioutil.ReadAll(writer)
	if err != nil {
		t.Fatal(err)
	}

	if !bytesEqual(actual, expected) {
		t.Errorf("read bytes != expected")
	}

}

func TestSlowReaderError(t *testing.T) {
	src, err := ioutil.ReadFile("./readwriterat.go")
	if err != nil {
		t.Fatal(err)
	}
	concurrency := 20
	partsize := int64(len(src) / concurrency)
	//expected := src[:int64(concurrency)*partsize]

	cw := chunkWriter{
		src:         src,
		concurrency: concurrency,
		partsize:    partsize,
	}

	writer := New()
	writer.Debug = true
	writer.PartSize = partsize

	expected := fmt.Errorf("test error")
	go func() {
		cw.WriteChunks(writer)
		writer.CloseWithError(expected)
	}()

	if _, err := ioutil.ReadAll(writer); err != expected {
		t.Fatal("expected error")
	}

}
