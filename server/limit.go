package main

import (
	"context"

	"golang.org/x/time/rate"
)

type simpleLimiter struct {
	limiter *rate.Limiter
}

func (l *simpleLimiter) Limit(_ context.Context) error {
	// if !l.limiter.Allow() {
	// 	return fmt.Errorf("reached rate-limiting %v", l.limiter.Limit())
	// }
	return nil
}
