package microtools

import (
	"time"
)

type SelectOptions struct {
	MaxLatency time.Duration
	CallBack   SelectorCallBack
}

type SelectorCallBack func(addr string)

// Option used to initialise the selector
type SelectOption func(*SelectOptions)

// MaxLatency sets the max latency used by the low latency selector
func MaxLatency(latency time.Duration) SelectOption {
	return func(o *SelectOptions) {
		o.MaxLatency = latency
	}
}

// CallBack sets the callback used by the low latency selector
func CallBack(callback SelectorCallBack) SelectOption {
	return func(o *SelectOptions) {
		o.CallBack = callback
	}
}
