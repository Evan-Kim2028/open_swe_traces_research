package unionstore

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipelinedConcurrentSet(t *testing.T) {
	p := NewPipelinedMemDB(emptyBufferBatchGetter, func(uint64, *MemDB) error { return nil })
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				key := []byte{byte(id), byte(j)}
				require.Nil(t, p.Set(key, key))
			}
		}(i)
	}
	wg.Wait()
	require.Equal(t, 8*200, p.memDB.Len())
}
