package main

import (
	"PPMA_compressor/internal/pkg/arithmetic_encoder_decoder"
	"PPMA_compressor/internal/pkg/compressor_decompressor"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	sourceFilePath := flag.String("f", "", "sourceFilePath. Should not be blank. if not exists - error")
	targetFilePath := flag.String("t", "", "targetFilePath - path to compressed sourceFile to be saved at, should not be blank. If exists - rewrite, if not exits - new")
	maxContextOrder := flag.Int("max-context-order", 4, "max context order, optional, default: 4")
	writerBufferSize := flag.Int("buffer-size", 4*1024*1024, "buffer size, optional, default: 4MB")

	flag.Parse()

	if *sourceFilePath == "" || *targetFilePath == "" {
		flag.Usage()
	}

	sourceFile, err := os.Open(*sourceFilePath)
	if err != nil {
		fmt.Printf("Error opening source sourceFile: %v\n", err)

		return
	}
	defer func() { _ = sourceFile.Close() }()

	targetFile, err := os.Create(*targetFilePath)
	if err != nil {
		fmt.Printf("Error creating target file: %v\n", err)

		return
	}
	defer func() { _ = targetFile.Close() }()

	bufTargetFile := bufio.NewWriterSize(targetFile, *writerBufferSize)

	arithmeticEncoder := arithmetic_encoder_decoder.NewArithmeticEncoder()

	compressor := compressor_decompressor.NewCompressor(bufTargetFile, arithmeticEncoder, *maxContextOrder)

	defer func() {
		if err := compressor.Close(); err != nil {
			fmt.Printf("Error closing compressor: %v\n", err)

			return
		}
		if err := bufTargetFile.Flush(); err != nil {
			fmt.Printf("Error flushing buffer: %v\n", err)
		}
	}()

	_, err = io.Copy(compressor, sourceFile)
	if err != nil {
		fmt.Printf("Error compressing sourceFile: %v\n", err)

		return
	}

	return
}

func GetFailToCompressError(err error) error {
	return fmt.Errorf("Error compressing sourceFile: %w", err)
}
