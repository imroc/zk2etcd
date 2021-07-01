package try

import "time"

func Do(fn func() bool, maxAttempts int, interval time.Duration) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	ok := false
	for i := 0; i < maxAttempts && !ok; i++ {
		ok = fn()
		time.Sleep(interval)
	}
}
