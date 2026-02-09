package cli

import (
	"fmt"
	"io"
)

type ProgressReader struct {
	reader     io.Reader
	total      int64
	current    int64
	onProgress func(int)
}

func NewProgressReader(reader io.Reader, total int64, onProgress func(int)) *ProgressReader {
	return &ProgressReader{reader: reader, total: total, onProgress: onProgress}
}

func (p *ProgressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.current += int64(n)
		p.report()
	}
	return n, err
}

func (p *ProgressReader) report() {
	if p.total > 0 {
		percent := int(float64(p.current) / float64(p.total) * 100)
		if p.onProgress != nil {
			p.onProgress(percent)
		}
		fmt.Printf("\r%d%%", percent)
		if percent >= 100 {
			fmt.Print("\n")
		}
		return
	}
	if p.onProgress != nil {
		p.onProgress(0)
	}
}
