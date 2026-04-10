package compressor_decompressor

import (
	"PPMC_compressor/internal/pkg/arithmetic_encoder_decoder"
	"PPMC_compressor/internal/pkg/context_tree"
	"PPMC_compressor/internal/pkg/sliding_window"
	"encoding/binary"
	"fmt"
	"io"
)

type Decompressor struct {
	decoder       *arithmetic_encoder_decoder.ArithmeticDecoder
	contextTree   *context_tree.ContextTree
	maxOrder      int
	slidingWindow *sliding_window.SlidingWindow
	remaining     uint64
	originalSize  uint64
	contextBuf    []byte // переиспользуемый буфер для контекста
}

// NewDecompressor читает заголовок из r и создаёт декомпрессор с арифметическим декодером.
func NewDecompressor(r io.Reader) (*Decompressor, error) {
	header := make([]byte, 9)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}
	maxOrder := int(header[0])
	originalSize := binary.LittleEndian.Uint64(header[1:])

	decoder := arithmetic_encoder_decoder.NewArithmeticDecoder(r)

	return &Decompressor{
		decoder:       decoder,
		contextTree:   context_tree.NewContextTree(maxOrder),
		maxOrder:      maxOrder,
		slidingWindow: sliding_window.NewSlidingWindow(maxOrder),
		remaining:     originalSize,
		originalSize:  originalSize,
		contextBuf:    make([]byte, maxOrder), // буфер для контекста
	}, nil
}

// OriginalSize возвращает исходный размер данных.
func (d *Decompressor) OriginalSize() uint64 {
	return d.originalSize
}

// Read реализует io.Reader – декомпрессия данных.
func (d *Decompressor) Read(p []byte) (n int, err error) {
	for n < len(p) && d.remaining > 0 {
		sym, err := d.decodeNextSymbol()
		if err != nil {
			return n, fmt.Errorf("failed to decode symbol: %w", err)
		}
		p[n] = byte(sym)
		n++
		d.remaining--

		// обновление модели для всех суффиксов контекста
		for o := d.maxOrder; o >= 0; o-- {
			ctx := d.slidingWindow.GetContext(o, d.contextBuf[:0])
			d.contextTree.Update(byte(sym), ctx)
		}
		d.slidingWindow.Push(byte(sym))
	}

	if d.remaining == 0 && n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// decodeNextSymbol декодирует один символ (0..255) из арифметического потока.
// Возвращает символ или ошибку.
func (d *Decompressor) decodeNextSymbol() (int, error) {
	order := d.maxOrder
	context := d.slidingWindow.GetContext(order, d.contextBuf[:0])

	for order >= 0 {
		node := d.contextTree.GetNode(context)

		if node != nil && len(node.Freq) > 0 {
			// Узел существует и имеет статистику
			escapeFreq := uint64(len(node.Freq))
			cum, total := GetCumFreqWithEscape(node.Freq, escapeFreq)

			sym, err := d.decoder.Decode(cum, total)
			PutCumFreq(cum)

			if err != nil {
				return 0, err
			}
			if sym != Escape {
				return sym, nil
			}
			// Escape – переходим к меньшему порядку
		} else {
			// Узел отсутствует или пуст – декодируем escape с частотой 1
			cum := make([]uint64, 258)
			cum[257] = 1
			sym, err := d.decoder.Decode(cum, 1)
			if err != nil {
				return 0, err
			}
			if sym != Escape {
				return 0, fmt.Errorf("expected Escape, got %d", sym)
			}
		}

		order--
		if order >= 0 {
			context = d.slidingWindow.GetContext(order, d.contextBuf[:0])
		}
	}

	// order == -1: равномерное распределение
	uniformCum, uniformTotal := GetUniformCumFreq()
	return d.decoder.Decode(uniformCum, uniformTotal)
}
