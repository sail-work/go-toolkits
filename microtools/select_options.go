package microtools

import (
	"time"
)

type SelectOptions struct {
	MaxLatency time.Duration
	Privileged bool
}

// Option used to initialise the selector
type SelectOption func(*SelectOptions)

// MaxLatency sets the max latency used by the low latency selector
func MaxLatency(latency time.Duration) SelectOption {
	return func(o *SelectOptions) {
		o.MaxLatency = latency
	}
}

// Privileged sets the ping mode for the low latency selector
func Privileged(privileged bool) SelectOption {
	return func(o *SelectOptions) {
		o.Privileged = privileged
	}
}
