package microtools

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
)

var (
	// DefaultMaxLatency is the default max latency
	DefaultMaxLatency = time.Second

	regex = `min/avg/max([\s\S]*)`
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
		nodes   = s.getNodes(services)
		timeout = s.maxLatency
		result  *registry.Node
		recv    chan interface{}
		err     error
	)

	for _, n := range nodes {
		if n.latency != 0 {
			timeout = n.latency
			result = n.n
			break
		}

		if recv == nil {
			recv = make(chan interface{})
		}
		go s.ping(n, recv)

	}
	if recv != nil {
		select {
		case <-time.After(timeout):
			err = fmt.Errorf("ping has timeout %s, %v", timeout, err)
		case v := <-recv:
			switch va := v.(type) {
			case *registry.Node:
				result = va
			default:
				err = fmt.Errorf("%s, %v", va, err)
			}
		}
	}

	return func() (*registry.Node, error) {
		if len(nodes) == 0 {
			return nil, selector.ErrNoneAvailable
		}

		if result == nil {
			if err != nil {
				return nil, err
			}
			return nil, errors.New("none result")
		}

		return result, nil
	}
}

func (s *lowLatencySelector) ping(node *node, recv chan interface{}) {
	var err error
	defer func() {
		var (
			v         interface{}
			blacklist bool
		)
		if err != nil {
			v = err
			blacklist = true
		} else {
			v = node.n
		}
		select {
		case recv <- v:
		default:
			// already lower latency
		}
		s.addNode(blacklist, node)
	}()

	host, _, err := net.SplitHostPort(node.n.Address)
	if err != nil {
		return
	}

	// ping only once
	cmd := exec.Command("ping", "-c", "1", host)
	b, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("[%s]%v", host, err)
		return
	}

	rtt, err := parsePing(string(b))
	if err != nil {
		err = fmt.Errorf("[%s]%v", host, err)
		return
	}

	if rtt > s.maxLatency {
		// if the maximum Latency is exceeded, add to the blacklist
		err = fmt.Errorf("[%s]maximum latency is exceeded", host)
		return
	}
	node.latency = rtt
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

func (s *lowLatencySelector) getNodes(services []*registry.Service) (nodes nodes) {
	var (
		diff bool
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
		// when the latency of the new node is greater than the lowest latency in the cache,
		// it will lead to no matching node, so clear cache
		for _, node := range nodes {
			node.latency = 0
		}
		s.mu.Lock()
		s.nodes = make(map[string]*node)
		s.blacklist = make(map[string]*node)
		s.mu.Unlock()
		return
	}
	sort.Sort(nodes)
	return
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
	if len(sub) < 2 {
		return 0, fmt.Errorf("parse ping error")
	}

	s := strings.Split(strings.TrimSpace(sub[0]), " ")
	unit := s[len(s)-1]
	times := strings.Split(s[len(s)-2], "/")
	if len(times) < 3 {
		return 0, fmt.Errorf("parse ping error")
	}
	return time.ParseDuration(fmt.Sprintf("%s%s", times[1], unit))
}
