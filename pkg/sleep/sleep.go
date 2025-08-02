package sleep

import (
	"context"
	"time"
)

func UntilContext(ctx context.Context, d time.Duration) bool {
	return Until(ctx.Done(), d)
}

func Until(ch <-chan struct{}, d time.Duration) bool {
	timer := time.NewTimer(d)
	select {
	case <-ch:
		if !timer.Stop() {
			<-timer.C
		}
		return false
	case <-timer.C:
		return true
	}
}
