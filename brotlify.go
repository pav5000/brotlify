package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/dustin/go-humanize"
	"github.com/pav5000/go-common/errors"
)

func skippedExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".webp", ".br", ".tar", ".gz", ".zip", ".mp3", ".ogg", ".mp4":
		return true
	}
	return false
}

type Brotlifier struct {
	totalSrc uint64
	totalDst uint64
}

func NewBrotlifier() *Brotlifier {
	return &Brotlifier{}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: brotlify <folder>")
		os.Exit(1)
	}
	br := NewBrotlifier()
	err := br.brotlify(os.Args[1])
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func (b *Brotlifier) brotlify(path string) error {
	fmt.Println("brotlifying", path)

	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Println(path)
			fmt.Println("   error:", err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		return b.processFile(path)
	})
}

func (b *Brotlifier) processFile(path string) error {
	name := filepath.Base(path)
	dir := filepath.Dir(path)
	ext := filepath.Ext(name)
	if skippedExt(ext) {
		return nil
	}
	fmt.Println(dir, name)

	rawData, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrp(err, "os.ReadFile")
	}

	compressedData := bytes.NewBuffer(nil)
	w := brotli.NewWriterLevel(compressedData, brotli.BestCompression)
	_, err = w.Write(rawData)
	if err != nil {
		return errors.Wrp(err, "compressing brotli")
	}

	err = w.Close()
	if err != nil {
		return errors.Wrp(err, "closing brotli")
	}

	srcSize := uint64(len(rawData))
	dstSize := uint64(len(compressedData.Bytes()))

	var ratio float64
	if dstSize != 0 {
		ratio = float64(dstSize) / float64(srcSize)
	}

	fmt.Printf(
		"    %s -> %s   %.0f%%\n",
		humanize.Bytes(srcSize),
		humanize.Bytes(dstSize),
		ratio*100,
	)

	if dstSize >= srcSize {
		fmt.Println("    skipped")
		b.totalSrc += srcSize
		b.totalDst += srcSize
		return nil
	}

	b.totalSrc += srcSize
	b.totalDst += dstSize

	compressedName := filepath.Join(dir, name+".br")
	err = os.WriteFile(compressedName, compressedData.Bytes(), 0o664)
	if err != nil {
		return errors.Wrp(err, "os.WriteFile")
	}

	return nil
}
