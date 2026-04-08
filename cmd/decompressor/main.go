package main

import (
	"PPMA_compressor/internal/pkg/compressor_decompressor"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/schollz/progressbar/v3"
)

type progressWriter struct {
	w   io.Writer
	bar *progressbar.ProgressBar
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.w.Write(p)
	_ = pw.bar.Add(n)
	return
}

func (pw *progressWriter) Finish() {
	_ = pw.bar.Finish()
}

func getMemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("%.1f MB", float64(m.Alloc)/1024/1024)
}

func main() {
	sourceFilePath := flag.String("f", "", "compressed source file path")
	targetFilePath := flag.String("t", "", "decompressed target file path")
	showProgress := flag.Bool("progress", true, "show progress bar")
	flag.Parse()

	if *sourceFilePath == "" || *targetFilePath == "" {
		flag.Usage()
		os.Exit(1)
	}

	sourceFile, err := os.Open(*sourceFilePath)
	if err != nil {
		fmt.Printf("Error opening source file: %v\n", err)
		return
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			fmt.Printf("Error closing source file: %v\n", err)
		}
	}()

	decompressor, err := compressor_decompressor.NewDecompressor(sourceFile)
	if err != nil {
		fmt.Printf("Error creating decompressor: %v\n", err)
		return
	}

	targetFile, err := os.Create(*targetFilePath)
	if err != nil {
		fmt.Printf("Error creating target file: %v\n", err)
		return
	}
	defer func() {
		if err := targetFile.Close(); err != nil {
			fmt.Printf("Error closing target file: %v\n", err)
		}
	}()

	var writer io.Writer = targetFile
	var pw *progressWriter
	if *showProgress {
		originalSize := decompressor.OriginalSize()
		if originalSize > 0 {
			bar := progressbar.DefaultBytes(int64(originalSize), "decompressing")
			pw = &progressWriter{w: targetFile, bar: bar}
			writer = pw
		}
	}

	// Запускаем горутину для обновления описания прогресс-бара с использованием памяти
	var stopCh chan struct{}
	if pw != nil {
		stopCh = make(chan struct{})
		defer close(stopCh)
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stopCh:
					return
				case <-ticker.C:
					pw.bar.Describe(fmt.Sprintf("decompressing (mem: %s)", getMemUsage()))
				}
			}
		}()
	}

	bufWriter := bufio.NewWriterSize(writer, 4*1024*1024)
	defer func() {
		if err := bufWriter.Flush(); err != nil {
			fmt.Printf("Error flushing buffer: %v\n", err)
		}
	}()

	_, err = io.Copy(bufWriter, decompressor)
	if pw != nil {
		pw.Finish()
	}
	if err != nil {
		fmt.Printf("Error decompressing: %v\n", err)
		return
	}

	fmt.Println("\nDecompression finished successfully")
}
