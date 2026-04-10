package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "E2E test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("E2E test passed")
}

func run() error {
	tmpDir, err := os.MkdirTemp("", "e2e_test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	original := filepath.Join(tmpDir, "original")
	compressed := filepath.Join(tmpDir, "compressed.ppmc")
	decompressed := filepath.Join(tmpDir, "decompressed")

	const size = 1 << 20
	if err := generateFile(original, size); err != nil {
		return err
	}

	compressBin := "./bin/ppmc_compress"
	decompressBin := "./bin/ppmc_decompress"

	cmdComp := exec.Command(compressBin, "-f", original, "-t", compressed)
	if out, err := cmdComp.CombinedOutput(); err != nil {
		return fmt.Errorf("compression: %w\n%s", err, out)
	}

	cmdDecomp := exec.Command(decompressBin, "-f", compressed, "-t", decompressed)
	if out, err := cmdDecomp.CombinedOutput(); err != nil {
		return fmt.Errorf("decompression: %w\n%s", err, out)
	}

	if !filesEqual(original, decompressed) {
		return fmt.Errorf("files differ")
	}
	return nil
}

func generateFile(path string, size int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	pattern := make([]byte, 256)
	for i := range pattern {
		pattern[i] = byte(i)
	}
	var written int64
	for written < size {
		n, err := f.Write(pattern)
		if err != nil {
			return err
		}
		written += int64(n)
	}
	return nil
}

func filesEqual(a, b string) bool {
	fa, _ := os.Open(a)
	defer fa.Close()
	fb, _ := os.Open(b)
	defer fb.Close()
	bufA := make([]byte, 4096)
	bufB := make([]byte, 4096)
	for {
		na, errA := fa.Read(bufA)
		nb, errB := fb.Read(bufB)
		if na != nb || !bytes.Equal(bufA[:na], bufB[:nb]) {
			return false
		}
		if errA == io.EOF && errB == io.EOF {
			return true
		}
		if errA != nil || errB != nil {
			return false
		}
	}
}
