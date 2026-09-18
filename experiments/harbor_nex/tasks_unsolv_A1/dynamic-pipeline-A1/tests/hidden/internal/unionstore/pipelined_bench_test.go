package unionstore

import (
	"context"
	"encoding/binary"
	"testing"
)

func BenchmarkPipelinedGet(b *testing.B) {
	ctx := context.Background()
	p := NewPipelinedMemDB(emptyBufferBatchGetter, func(uint64, *MemDB) error { return nil })
	buf := make([][16]byte, 5000)
	for i := range buf {
		binary.BigEndian.PutUint32(buf[i][:], uint32(i))
		if err := p.Set(buf[i][:], buf[i][:]); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = p.Get(ctx, buf[i%len(buf)][:])
			i++
		}
	})
}
