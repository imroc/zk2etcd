package try

import "time"

func Do(fn func() bool, maxAttempts int, interval time.Duration) {

	ok := fn()
	if ok { // 一次就成功，无需重试
		return
	}

	if maxAttempts <= 0 { // 无限重试
		for !ok {
			time.Sleep(interval)
			ok = fn()
		}
		return
	}

	for i := 1; i <= maxAttempts && !ok; i++ { // 重试不超过最大次数
		time.Sleep(interval)
		ok = fn()
	}
}
