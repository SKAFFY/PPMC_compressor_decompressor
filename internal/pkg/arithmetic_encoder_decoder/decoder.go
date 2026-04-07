package arithmetic_encoder_decoder

import (
	"fmt"
	"io"
)

type ArithmeticDecoder struct {
	low, high uint64
	value     uint64
	in        io.Reader
	buf       byte
	bits      uint8
	err       error
}

func NewArithmeticDecoder(r io.Reader) *ArithmeticDecoder {
	d := &ArithmeticDecoder{
		low:  0,
		high: TopValue - 1,
		in:   r,
	}
	for i := 0; i < CodeValueBits; i++ {
		bit := d.readBit()

		d.value = (d.value << 1) | bit
	}
	return d
}

func (d *ArithmeticDecoder) readBit() uint64 {
	if d.bits == 0 {
		var b [1]byte
		n, err := d.in.Read(b[:])
		if n == 1 {
			d.buf = b[0]
			d.bits = 8
		} else {
			// поток закончился -> бесконечные нули
			d.buf = 0
			d.bits = 8
		}
		_ = err
	}
	d.bits--

	return uint64((d.buf >> d.bits) & 1)
}

func (d *ArithmeticDecoder) Decode(cumFreq []uint64, totalFreq uint64) (byte, error) {
	if d.err != nil {
		return 0, fmt.Errorf("decode previous err exists: %w", io.ErrUnexpectedEOF)
	}
	rng := d.high - d.low + 1
	scaled := ((d.value-d.low+1)*totalFreq - 1) / rng

	lo, hi := 0, len(cumFreq)-2
	for lo <= hi {
		mid := (lo + hi) / 2
		if cumFreq[mid] <= scaled && scaled < cumFreq[mid+1] {
			sym := byte(mid)
			d.high = d.low + (rng*cumFreq[sym+1])/totalFreq - 1
			d.low = d.low + (rng*cumFreq[sym])/totalFreq

			for {
				if d.high < Half {
					// бит не меняется
				} else if d.low >= Half {
					d.value -= Half
					d.low -= Half
					d.high -= Half
				} else if d.low >= FirstQtr && d.high < ThirdQtr {
					d.value -= FirstQtr
					d.low -= FirstQtr
					d.high -= FirstQtr
				} else {
					break
				}
				d.low <<= 1
				d.high = (d.high << 1) | 1
				bit := d.readBit()

				d.value = (d.value << 1) | bit
			}
			return sym, nil
		} else if scaled < cumFreq[mid] {
			hi = mid - 1
		} else {
			lo = mid + 1
		}
	}
	return 0, fmt.Errorf("decode end: %w", io.ErrUnexpectedEOF)
}
