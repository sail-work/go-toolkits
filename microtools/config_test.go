package microtools

import (
	"testing"
	"time"

	"github.com/sparrc/go-ping"
)

type kv struct {
	Data string
}

func Test_Put(t *testing.T) {
	InitSource(WithFrom("consul://kvTest"))

	err := Put(&kv{
		Data: "put test",
	}, "123")
	if err != nil {
		t.Error(err)
	}
}

func Test_ConfigGet(t *testing.T) {
	type args struct {
		x    interface{}
		path []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ConfigGet(tt.args.x, tt.args.path...); (err != nil) != tt.wantErr {
				t.Errorf("ConfigGet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_(t *testing.T) {
	start := time.Now()
	p, _ := ping.NewPinger("127.0.0.1")
	p.SetPrivileged(false)
	p.Count = 1
	p.OnRecv = func(packet *ping.Packet) {
		t.Logf("rtt:%s", packet.Rtt)
	}

	p.Run()
	t.Logf("time:%s", time.Since(start))
}
