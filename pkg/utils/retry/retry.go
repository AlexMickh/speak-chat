package retry

import "time"

func WithDelay(maxAttempts int, delay time.Duration, fn func() error) error {
	var err error

	for maxAttempts > 0 {
		if err = fn(); err != nil {
			time.Sleep(delay)
			maxAttempts--
			continue
		}

		return nil
	}

	return err
}
