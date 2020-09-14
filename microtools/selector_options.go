package microtools

import (
	"time"
)

type SelectOptions struct {
	MaxLatency time.Duration
}

// Option used to initialise the selector
type SelectOption func(*SelectOptions)

// MaxLatency sets the max latency used by the low latency selector
func MaxLatency(latency time.Duration) SelectOption {
	return func(o *SelectOptions) {
		o.MaxLatency = latency
	}
}
