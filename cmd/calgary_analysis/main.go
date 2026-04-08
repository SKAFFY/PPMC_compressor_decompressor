package main

import (
	"PPMA_compressor/internal/pkg/arithmetic_encoder_decoder"
	"PPMA_compressor/internal/pkg/compressor_decompressor"
	"bytes"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

// entropyZeroOrder вычисляет энтропию 0-го порядка (бит на символ)
func entropyZeroOrder(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}
	freq := make([]int, 256)
	for _, b := range data {
		freq[b]++
	}
	total := float64(len(data))
	var h float64
	for _, f := range freq {
		if f == 0 {
			continue
		}
		p := float64(f) / total
		h -= p * math.Log2(p)
	}
	return h
}

// entropyFirstOrder вычисляет условную энтропию H(X_{i+1} | X_i)
func entropyFirstOrder(data []byte) float64 {
	if len(data) < 2 {
		return 0.0
	}
	// freq отдельных символов
	freqSym := make([]int, 256)
	// freq пар (prev, next) – упакуем в int = prev*256 + next
	pairFreq := make([]int, 256*256)
	for i := 0; i < len(data)-1; i++ {
		prev := data[i]
		next := data[i+1]
		freqSym[prev]++
		pairFreq[int(prev)<<8|int(next)]++
	}
	// последний символ не имеет следующего, но для p(prev) нужно учесть все символы, кроме последнего?
	// Для условной энтропии H(Y|X) сумма по X p(X) * H(Y|X). p(X) = count(X) / (len(data)-1) – количество пар, где X выступает первым.
	// Суммируем по всем парам: -p(prev,next) * log2(p(next|prev))
	totalPairs := len(data) - 1
	var h float64
	for prev := 0; prev < 256; prev++ {
		countPrev := freqSym[prev]
		if countPrev == 0 {
			continue
		}
		// p(prev) = countPrev / totalPairs
		for next := 0; next < 256; next++ {
			cnt := pairFreq[prev<<8|next]
			if cnt == 0 {
				continue
			}
			pPair := float64(cnt) / float64(totalPairs)
			pCond := float64(cnt) / float64(countPrev)
			h -= pPair * math.Log2(pCond)
		}
	}
	return h
}

// entropySecondOrder вычисляет условную энтропию H(X_{i+2} | X_i, X_{i+1})
func entropySecondOrder(data []byte) float64 {
	if len(data) < 3 {
		return 0.0
	}
	// freq контекстов из двух символов (prev1, prev2) – упакуем в int = (prev1<<8)|prev2
	contextFreq := make([]int, 256*256)
	// freq троек (ctx, next) – упакуем в int = (ctx<<8)|next, где ctx = (prev1<<8)|prev2
	tripleFreq := make([]int, 256*256*256) // 16 млн – 16*8=128MB, допустимо для небольших файлов
	for i := 0; i < len(data)-2; i++ {
		ctx := int(data[i])<<8 | int(data[i+1])
		next := data[i+2]
		contextFreq[ctx]++
		tripleFreq[ctx<<8|int(next)]++
	}
	totalTriples := len(data) - 2
	var h float64
	for ctx := 0; ctx < 256*256; ctx++ {
		countCtx := contextFreq[ctx]
		if countCtx == 0 {
			continue
		}
		for next := 0; next < 256; next++ {
			cnt := tripleFreq[ctx<<8|next]
			if cnt == 0 {
				continue
			}
			pTriple := float64(cnt) / float64(totalTriples)
			pCond := float64(cnt) / float64(countCtx)
			h -= pTriple * math.Log2(pCond)
		}
	}
	return h
}

// compressedSizeAndBPC сжимает данные с помощью PPM (maxOrder=4) и возвращает размер сжатых данных в байтах и средние байты на символ.
func compressedSizeAndBPC(data []byte, maxOrder int) (compSize int, bpc float64) {
	var buf bytes.Buffer
	enc := arithmetic_encoder_decoder.NewArithmeticEncoder(&buf)
	comp, err := compressor_decompressor.NewCompressor(&buf, enc, maxOrder, uint64(len(data)))
	if err != nil {
		return 0, 0
	}
	_, err = comp.Write(data)
	if err != nil {
		return 0, 0
	}
	err = comp.Close()
	if err != nil {
		return 0, 0
	}
	compSize = buf.Len()
	if len(data) > 0 {
		bpc = float64(compSize) / float64(len(data))
	}
	return compSize, bpc
}

func main() {
	dir := flag.String("dir", "../test/test_dataset", "Directory containing Calgary corpus files (e.g., bib, book1, book2, geo, news, obj1, obj2, paper1, paper2, pic, progc, progl, progp, trans)")
	maxOrder := flag.Int("max-order", 4, "Max context order for PPM compressor")
	outputFile := flag.String("output", "", "Save table to file (CSV format, append .csv or .md)")
	flag.Parse()
	if *dir == "" {
		fmt.Println("Please provide -dir with path to Calgary corpus files")
		flag.Usage()
		os.Exit(1)
	}

	// Список файлов Calgary corpus (14 имён)
	fileNames := []string{
		"bib", "book1", "book2", "geo", "news", "obj1", "obj2",
		"paper1", "paper2", "pic", "progc", "progl", "progp", "trans",
	}

	var out io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot create output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = io.MultiWriter(os.Stdout, f)
	}

	fmt.Fprintf(out, "%-10s %8s %8s %8s %10s %12s %12s\n",
		"File", "H0 (bits)", "H1 (bits)", "H2 (bits)", "Size (B)", "CompSize (B)", "BPC (B/sym)")
	fmt.Fprintf(out, "---------- -------- -------- -------- ---------- ------------ ------------\n")

	var totalCompSize int64 = 0

	for _, name := range fileNames {
		path := filepath.Join(*dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(out, "%-10s %s\n", name, fmt.Sprintf("ERROR: %v", err))
			continue
		}
		h0 := entropyZeroOrder(data)
		h1 := entropyFirstOrder(data)
		h2 := entropySecondOrder(data)
		compSize, bpc := compressedSizeAndBPC(data, *maxOrder)
		totalCompSize += int64(compSize)

		fmt.Fprintf(out, "%-10s %8.4f %8.4f %8.4f %10d %12d %12.4f\n",
			name, h0, h1, h2, len(data), compSize, bpc)
	}
	fmt.Fprintf(out, "\nTotal compressed size (all files): %d bytes\n", totalCompSize)
}
