package arithmetic_encoder_decoder

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uniformCumFreq() ([]uint64, uint64) {
	cum := make([]uint64, 257)
	for i := 0; i < 256; i++ {
		cum[i+1] = cum[i] + 1
	}
	return cum, 256
}

func TestEncoderDecoder(t *testing.T) {
	tests := []struct {
		name    string
		symbols []byte
	}{
		{"single symbol", []byte{'A'}},
		{"two symbols", []byte{0, 1, 0, 1, 0}},
		{"three symbols", []byte{0, 1, 2, 0, 2, 1, 0}},
		{"hello world", []byte("Hello, world!")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewArithmeticEncoder(&buf)
			cum, total := uniformCumFreq()

			for _, sym := range tt.symbols {
				enc.Encode(sym, cum, total)
			}
			err := enc.Flush()
			require.NoError(t, err)

			t.Logf("Compressed length: %d bytes", buf.Len())
			if buf.Len() == 0 {
				t.Fatal("No data written")
			}

			dec := NewArithmeticDecoder(&buf)

			decoded := make([]byte, len(tt.symbols))
			for i := 0; i < len(tt.symbols); i++ {
				sym, err := dec.Decode(cum, total)
				if err != nil {
					t.Logf("i: %d err: %v", i, err)
				}
				decoded[i] = sym
			}
			assert.Equal(t, tt.symbols, decoded)
		})
	}
}
