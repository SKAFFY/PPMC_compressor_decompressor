package compressor_decompressor

import (
	"PPMC_compressor/internal/pkg/context_tree"
	"PPMC_compressor/internal/pkg/sliding_window"
	"encoding/binary"
	"io"
)

// EncoderWriter определяет интерфейс арифметического кодера
type EncoderWriter interface {
	Encode(sym int, cumFreq []uint64, totalFreq uint64)
	Flush() error
}

type Compressor struct {
	encoder       EncoderWriter
	contextTree   *context_tree.ContextTree
	maxOrder      int
	slidingWindow *sliding_window.SlidingWindow
	contextBuf    []byte // переиспользуемый буфер для контекста
}

// NewCompressor принимает writer (для заголовка) и энкодер (который пишет в тот же writer)
func NewCompressor(w io.Writer, encoder EncoderWriter, maxOrder int, originalSize uint64) (*Compressor, error) {
	// Заголовок: 1 байт maxOrder + 8 байт originalSize
	header := make([]byte, 9)
	header[0] = byte(maxOrder)
	binary.LittleEndian.PutUint64(header[1:], originalSize)
	if _, err := w.Write(header); err != nil {
		return nil, err
	}
	return &Compressor{
		encoder:       encoder,
		contextTree:   context_tree.NewContextTree(maxOrder),
		maxOrder:      maxOrder,
		slidingWindow: sliding_window.NewSlidingWindow(maxOrder),
		contextBuf:    make([]byte, maxOrder), // буфер для контекста
	}, nil
}

// Write реализует io.Writer – сжимает поступающие данные и передаёт в арифметический кодер
func (c *Compressor) Write(p []byte) (n int, err error) {
	for _, sym := range p {
		order := c.maxOrder
		// получаем контекст максимального порядка, используя переиспользуемый буфер
		context := c.slidingWindow.GetContext(order, c.contextBuf[:0])

		for order >= 0 {
			node := c.contextTree.GetNode(context)
			if node != nil && node.Freq[sym] > 0 {
				// Символ найден – кодируем его, используя распределение с escape
				escapeFreq := uint64(len(node.Freq)) // метод C
				if escapeFreq == 0 {
					escapeFreq = 1
				}
				cum, total := GetCumFreqWithEscape(node.Freq, escapeFreq)
				c.encoder.Encode(int(sym), cum, total)
				PutCumFreq(cum)
				break
			} else {
				// Символ не найден – кодируем escape и переходим к меньшему порядку
				escapeFreq := uint64(1)
				if node != nil {
					escapeFreq = uint64(len(node.Freq))
					if escapeFreq == 0 {
						escapeFreq = 1
					}
				}
				if node != nil {
					cum, total := GetCumFreqWithEscape(node.Freq, escapeFreq)
					c.encoder.Encode(Escape, cum, total)
					PutCumFreq(cum)
				} else {
					// узел не существует – только escape с частотой 1
					cum := make([]uint64, 258)
					cum[257] = escapeFreq
					c.encoder.Encode(Escape, cum, escapeFreq)
				}

				order--
				if order >= 0 {
					context = c.slidingWindow.GetContext(order, c.contextBuf[:0])
				}
				continue
			}
		}

		if order < 0 {
			// Порядок -1: равномерное распределение по 256 символам
			uniformCum, uniformTotal := GetUniformCumFreq()
			c.encoder.Encode(int(sym), uniformCum, uniformTotal)
		}

		// Обновляем статистику для всех суффиксов контекста (от maxOrder до 0)
		for o := c.maxOrder; o >= 0; o-- {
			ctx := c.slidingWindow.GetContext(o, c.contextBuf[:0])
			c.contextTree.Update(sym, ctx)
		}
		c.slidingWindow.Push(sym)
	}
	return len(p), nil
}

// Close завершает сжатие – вызывает Flush у арифметического кодера
func (c *Compressor) Close() error {
	return c.encoder.Flush()
}
