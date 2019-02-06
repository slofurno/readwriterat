### readwriterat

when a library expects a WriterAt, but you would like to consume it as a stream

made specifically for aws s3manager downloader, but possibly useful in other places

in worst case, will buffer up to Concurrency * 2 * PartSize

```go

customPartSize := 64 * 1024 * 1024 // 64MB per part

downloader := s3manager.NewDownloader(cfg, func(d *s3manager.Downloader) {
  d.PartSize = customPartSize
})

//configured by default to match aws default part size
rw := readwriterat.New(func(rw *readwriterat.ReadWriterAt) {
  rw.PartSize = customPartSize
})

```
