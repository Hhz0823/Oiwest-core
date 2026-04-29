package buf

import (
	"io"
	"sync"
)

const (
	Size    = 2048
	MaxSize = 64 * 1024
)

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, Size)
	},
}

type Buffer struct {
	buf    []byte
	start  int
	end    int
	pooled bool
}

func New() *Buffer {
	return &Buffer{
		buf:    pool.Get().([]byte),
		pooled: true,
	}
}

func NewWithSize(size int) *Buffer {
	if size <= Size {
		return New()
	}
	return &Buffer{
		buf: make([]byte, size),
	}
}

func StackNew() Buffer {
	return Buffer{
		buf: make([]byte, Size),
	}
}

func (b *Buffer) Byte(idx int) byte {
	return b.buf[b.start+idx]
}

func (b *Buffer) Bytes() []byte {
	return b.buf[b.start:b.end]
}

func (b *Buffer) BytesFrom(idx int) []byte {
	return b.buf[b.start+idx : b.end]
}

func (b *Buffer) BytesRange(from, to int) []byte {
	return b.buf[b.start+from : b.start+to]
}

func (b *Buffer) Len() int {
	return b.end - b.start
}

func (b *Buffer) IsEmpty() bool {
	return b.start == b.end
}

func (b *Buffer) IsFull() bool {
	return b.end == len(b.buf)
}

func (b *Buffer) Clear() {
	b.start = 0
	b.end = 0
}

func (b *Buffer) Reset(fn func([]byte) int) int {
	index := fn(b.buf)
	b.start = 0
	b.end = index
	return index
}

func (b *Buffer) Advance(from int) {
	b.start += from
}

func (b *Buffer) Extend(to int) int {
	extLen := to
	nBytes := copy(b.buf[b.end:], b.buf[b.start:b.start+extLen])
	b.end += nBytes
	return nBytes
}

func (b *Buffer) Write(data []byte) (int, error) {
	n := copy(b.buf[b.end:], data)
	b.end += n
	if n < len(data) {
		return n, io.ErrShortBuffer
	}
	return n, nil
}

func (b *Buffer) WriteByte(d byte) error {
	if b.IsFull() {
		return io.ErrShortBuffer
	}
	b.buf[b.end] = d
	b.end++
	return nil
}

func (b *Buffer) WriteBytes(d ...byte) (int, error) {
	return b.Write(d)
}

func (b *Buffer) Read(data []byte) (int, error) {
	if b.IsEmpty() {
		return 0, io.EOF
	}
	n := copy(data, b.buf[b.start:b.end])
	b.start += n
	return n, nil
}

func (b *Buffer) ReadByte() (byte, error) {
	if b.IsEmpty() {
		return 0, io.EOF
	}
	v := b.buf[b.start]
	b.start++
	return v, nil
}

func (b *Buffer) ReadFullFrom(reader io.Reader, size int) (int, error) {
	end := b.end + size
	if end > len(b.buf) {
		nb := make([]byte, end)
		copy(nb, b.buf[b.start:b.end])
		b.buf = nb
		b.start = 0
		b.end = end - size
		b.pooled = false
	}
	n, err := io.ReadFull(reader, b.buf[b.end:end])
	b.end += n
	return n, err
}

func (b *Buffer) Release() {
	if b == nil || !b.pooled {
		return
	}
	b.start = 0
	b.end = 0
	pool.Put(b.buf)
}

func (b *Buffer) Slice(from, to int) *Buffer {
	return &Buffer{
		buf:   b.buf,
		start: b.start + from,
		end:   b.start + to,
	}
}

func (b *Buffer) Clone() *Buffer {
	nb := NewWithSize(b.Len())
	nb.end = copy(nb.buf, b.Bytes())
	return nb
}

func (b *Buffer) Resize(start, end int) {
	b.start = b.start + start
	b.end = b.start + end
}
