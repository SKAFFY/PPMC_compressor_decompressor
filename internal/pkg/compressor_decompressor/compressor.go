package compressor_decompressor

import (
	"PPMA_compressor/internal/pkg/context_tree"
	"io"
)

const (
	Escape int = 1
)

type Encoder interface {
	Encode(sym byte, cumFreq []uint32, totalFreq uint32)
	Flush() []byte
}

type Compressor struct {
	dst io.Writer

	encoder       Encoder
	contextTree   *context_tree.ContextTree
	maxOrder      int
	slidingWindow *slidingWindow
}

func NewCompressor(dst io.Writer, encoder Encoder, maxOrder int) *Compressor {
	return &Compressor{
		dst:           dst,
		encoder:       encoder,
		contextTree:   context_tree.NewContextTree(maxOrder),
		maxOrder:      maxOrder,
		slidingWindow: NewSlidingWindow(maxOrder),
	}
}

func (c *Compressor) Write(p []byte) (n int, err error) {

}

func (c *Compressor) Close() error {
	//TODO implement me
	panic("implement me")
}

func (c *Compressor) compress(data []byte) []byte {

	for _, sym := range data {
		order := c.maxOrder
		context := c.slidingWindow.GetContext(order)

		for order >= 0 {
			probs := c.contextTree.GetProbabilities(context)

			if _, exists := probs[sym]; exists {
				c.encoder.EncodeSymbol(sym, probs)
				break
			} else {
				c.encoder.EncodeSymbol(Escape, probs)
				order--
				context = c.slidingWindow.GetContext(order)
			}
		}

		if order < 0 {
			c.encoder.EncodeSymbolUniform(sym, 256)
		}

		c.contextTree.Update(sym, c.slidingWindow.GetContext(c.maxOrder))
		c.slidingWindow.Push(sym)
	}

	return c.encoder.Flush()
}
