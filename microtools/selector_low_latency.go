package microtools

import (
	"sort"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
	"github.com/sparrc/go-ping"
)

type lowLatencySelector struct {
	selector   selector.Selector
	timeout    time.Duration
	nodes      map[string]*node
	privileged bool
	mu         sync.RWMutex
}

func LowLatencySelector(privileged bool, d time.Duration) client.Option {
	return func(o *client.Options) {
		s := o.Selector
		if s == nil {
			s = selector.DefaultSelector
		}
		o.Selector = &lowLatencySelector{
			selector:   s,
			nodes:      make(map[string]*node),
			timeout:    d,
			privileged: privileged,
		}
		o.Selector.Init()
	}
}

func (s *lowLatencySelector) Init(opts ...selector.Option) error {
	opts = append(opts, selector.SetStrategy(s.LowLatency))
	return s.selector.Init(opts...)
}

func (s *lowLatencySelector) Options() selector.Options {
	return s.selector.Options()
}

func (s *lowLatencySelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	return s.selector.Select(service, opts...)
}

func (s *lowLatencySelector) Mark(service string, node *registry.Node, err error) {
	s.selector.Mark(service, node, err)
}

func (s *lowLatencySelector) Reset(service string) {
}

// Close stops the watcher and destroys the cache
func (s *lowLatencySelector) Close() error {
	return s.selector.Close()
}

func (s *lowLatencySelector) String() string {
	return s.selector.String()
}

func (s *lowLatencySelector) LowLatency(services []*registry.Service) selector.Next {
	var (
		nodes   nodes
		lowest  *node
		latency = s.timeout
		result  *registry.Node
		recv    chan *registry.Node
		diff    bool
	)
	s.mu.RLock()
	for _, service := range services {
		for _, n := range service.Nodes {
			latency, ok := s.nodes[n.Id]
			if !ok {
				latency = &node{n: n}
				diff = true
			}
			nodes = append(nodes, latency)
		}
	}
	s.mu.RUnlock()

	if diff {
		s.mu.Lock()
		for id, n := range s.nodes {
			var exist bool
			for _, n2 := range nodes {
				if n2.n.Id == n.n.Id {
					exist = true
					break
				}
			}
			if !exist {
				delete(s.nodes, id)
			}
		}
		s.mu.Unlock()
	}

	sort.Sort(nodes)
	for _, n := range nodes {
		if n.latency != 0 {
			lowest = n
			result = lowest.n
			break
		}

		if recv == nil {
			recv = make(chan *registry.Node)
		}
		go s.ping(n, recv)

	}
	if recv != nil {
		if lowest != nil {
			latency = lowest.latency
		}
		select {
		case <-time.After(latency):
		case n := <-recv:
			result = n
		}
	}

	return func() (*registry.Node, error) {
		if len(nodes) == 0 {
			return nil, selector.ErrNoneAvailable
		}

		return result, nil
	}
}

func (s *lowLatencySelector) ping(node *node, recv chan *registry.Node) {
	p, err := ping.NewPinger(node.n.Address)
	if err != nil {
		node.latency = s.timeout
		s.addNode(node)
		return
	}
	p.SetPrivileged(s.privileged)
	p.Count = 1
	p.OnRecv = func(packet *ping.Packet) {
		node.latency = packet.Rtt
		select {
		case recv <- node.n:
		default:
		}
		s.addNode(node)
	}
	p.Run()
}

func (s *lowLatencySelector) addNode(node *node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node.n.Id] = node
}

type node struct {
	n       *registry.Node
	latency time.Duration
}

type nodes []*node

func (n nodes) Len() int { return len(n) }

func (n nodes) Less(i, j int) bool { return n[i].latency < n[j].latency }

func (n nodes) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
