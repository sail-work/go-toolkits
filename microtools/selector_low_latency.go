package microtools

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hkjojo/go-toolkits/log"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
)

var (
	// DefaultMaxLatency is the default max latency
	DefaultMaxLatency = time.Second

	regex = `min/avg/max/stddev = ([0-9./]+) ms`
)

type lowLatencySelector struct {
	mu         sync.RWMutex
	selector   selector.Selector
	maxLatency time.Duration
	nodes      map[string]*node
	blacklist  map[string]*node
}

func LowLatencySelector(opts ...SelectOption) client.Option {
	return func(o *client.Options) {
		selectOpt := &SelectOptions{
			MaxLatency: GetMaxLatency(),
		}

		for _, opt := range opts {
			opt(selectOpt)
		}

		s := o.Selector
		if s == nil {
			s = selector.DefaultSelector
		}
		o.Selector = &lowLatencySelector{
			nodes:      make(map[string]*node),
			blacklist:  make(map[string]*node),
			selector:   s,
			maxLatency: selectOpt.MaxLatency,
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
		latency = s.maxLatency
		result  *registry.Node
		recv    chan *registry.Node
		diff    bool
	)
	s.mu.RLock()
	for _, service := range services {
		for _, n := range service.Nodes {
			// check blacklist
			if _, ok := s.blacklist[n.Id]; ok {
				continue
			}

			// check cache
			cacheNode, ok := s.nodes[n.Id]
			if !ok {
				cacheNode = &node{n: n}
				diff = true
			}
			nodes = append(nodes, cacheNode)
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

		if result == nil {
			return nil, selector.ErrNoneAvailable
		}

		return result, nil
	}
}

func (s *lowLatencySelector) ping(node *node, recv chan *registry.Node) {
	host, _, err := net.SplitHostPort(node.n.Address)
	if err != nil {
		s.addNode(true, node)
		return
	}

	cmd := exec.Command("ping", "-c", "1", host)
	b, err := cmd.Output()
	if err != nil {
		log.Infof("ping err:%s", err)
		s.addNode(true, node)
		return
	}

	rtt, err := parsePing(string(b))
	if err != nil {
		log.Infof("parse ping err:%s", err)
		s.addNode(true, node)
		return
	}

	if rtt > s.maxLatency {
		// if the maximum Latency is exceeded, add to the blacklist
		s.addNode(true, node)
		return
	}
	node.latency = rtt
	select {
	case recv <- node.n:
	default:
		// already lower latency
		return
	}
	s.addNode(false, node)
}

func (s *lowLatencySelector) addNode(blacklist bool, node *node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if blacklist {
		s.blacklist[node.n.Id] = node
		return
	}
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

func parsePing(result string) (time.Duration, error) {
	re := regexp.MustCompile(regex)
	sub := re.FindStringSubmatch(result)
	if len(sub) != 2 {
		return 0, fmt.Errorf("parse ping error")
	}

	s := strings.Split(sub[0], " ")
	unit := s[len(s)-1]
	times := strings.Split(sub[1], "/")
	if len(times) != 4 {
		return 0, fmt.Errorf("parse ping error")
	}
	return time.ParseDuration(fmt.Sprintf("%s%s", times[1], unit))
}
