package arithmetic_encoder_decoder

import (
	"bytes"
	"testing"
)

func TestSimple(t *testing.T) {
	data := []byte("Hello, world!")
	var buf bytes.Buffer
	enc := NewArithmeticEncoder(&buf)
	cum, total := uniformCumFreq()
	for _, b := range data {
		enc.Encode(b, cum, total)
	}
	enc.Flush()
	t.Logf("Encoded %d bytes", buf.Len())

	dec := NewArithmeticDecoder(&buf)
	out := make([]byte, len(data))
	for i := range out {
		sym, err := dec.Decode(cum, total)
		if err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
		out[i] = sym
	}
	if string(out) != string(data) {
		t.Errorf("got %q, want %q", out, data)
	}
}
