package try

import "time"

func Do(fn func() bool, maxAttempts int, interval time.Duration) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	ok := fn()
	for i := 1; i <= maxAttempts && !ok; i++ {
		time.Sleep(interval)
		ok = fn()
	}
}
