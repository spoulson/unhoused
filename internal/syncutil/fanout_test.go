package syncutil_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"unhoused/internal/syncutil"
)

// You can make this test fail by messing with maxConcurrent
// in NewFanOut().
func TestFanOut(t *testing.T) {
	over := make(chan int, 10)
	var cur atomic.Int64

	f := syncutil.NewFanOut(5)
	for i := 0; i < 10; i++ {
		f.Run(func(a any) error {
			curr := cur.Add(1)
			if curr > 5 {
				over <- int(curr)
			}

			// Synthetic delay to catch concurrency overage
			time.Sleep(100 * time.Millisecond)
			cur.Add(-1)
			return nil
		}, i)
	}

	errs := f.Wait()
	require.Empty(t, errs, "FanOut had errors: %#v", errs)

	// We will get multiple logs noting how far over we went. Useful to debug
	select {
	case cur := <-over:
		t.Errorf("concurrency limit breached: count reached %d", cur)
	default:
		// No violations detected
	}
}
