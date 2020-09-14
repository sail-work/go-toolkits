package microtools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePingResult(t *testing.T) {
	pingMsg := map[string]string{
		"mac": `PING 127.0.0.1 (127.0.0.1): 56 data bytes
64 bytes from 127.0.0.1: icmp_seq=0 ttl=64 time=0.036 ms

--- 127.0.0.1 ping statistics ---
1 packets transmitted, 1 packets received, 0.0% packet loss
round-trip min/avg/max/stddev = 0.036/0.036/0.036/0.000 ms`,

		"linux": `PING 127.0.0.1 (127.0.0.1) 56(84) bytes of data.
64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=0.033 ms

--- 127.0.0.1 ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 0.033/0.033/0.033/0.000 ms`,

		"docker": `PING 127.0.0.1 (127.0.0.1): 56 data bytes
64 bytes from 127.0.0.1: seq=0 ttl=64 time=0.040 ms

--- 127.0.0.1 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 0.040/0.040/0.040 ms`,
	}

	for p, msg := range pingMsg {
		avg, err := parsePing(msg)
		assert.NoError(t, err)
		switch p {
		case "mac":
			assert.Equal(t, "36µs", avg.String())
		case "linux":
			assert.Equal(t, "33µs", avg.String())
		case "docker":
			assert.Equal(t, "40µs", avg.String())
		}
	}
}
