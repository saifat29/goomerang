package proxy

import (
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/cespare/xxhash/v2"

	"github.com/saifat29/goomerang/config"
)

// Balancer selects the most appropriate upstream URL based on the
// load balancing strategy.
type Balancer interface {
	Select(r *http.Request) *url.URL
}

// NewBalancer creates a new Balancer based on the configured
// strategy for an upstream.
func NewBalancer(cfg *config.Proxy) Balancer {
	switch cfg.Strategy {
	case config.StrategyIPHash:
		return NewIPHash(cfg.Upstreams)
	default:
		return NewWeightedRoundRobin(cfg.Upstreams)
	}
}

// WeightedRoundRobin implements the weighted round robin strategy
// for selecting an upstream server.
type WeightedRoundRobin struct {
	mu          sync.Mutex
	upstreams   []*weightedUpstream
	totalWeight int
}

// weightedUpstream is the upstream for which it's configured `weight`
// and `currentWeight` is tracked as per the algorithm.
type weightedUpstream struct {
	url           *url.URL
	weight        int
	currentWeight int
}

// NewWeightedRoundRobin returns the WeightedRoundRobin load balancer.
func NewWeightedRoundRobin(upstreams []*config.UpstreamServer) *WeightedRoundRobin {
	b := &WeightedRoundRobin{
		upstreams: make([]*weightedUpstream, len(upstreams)),
	}

	for i, upstream := range upstreams {
		weight := max(upstream.Weight, 1) // If not set, defaults to `1`.
		b.upstreams[i] = &weightedUpstream{
			url:           upstream.URL.URL,
			weight:        weight,
			currentWeight: 0,
		}
		b.totalWeight += weight
	}

	return b
}

// Select implements the Weighted Round Robin algorithm to select
// and return the chosen upstream URL.
func (b *WeightedRoundRobin) Select(*http.Request) *url.URL {
	b.mu.Lock()
	defer b.mu.Unlock()

	var best *weightedUpstream

	for _, upstream := range b.upstreams {
		upstream.currentWeight += upstream.weight
		if best == nil || upstream.currentWeight > best.currentWeight {
			best = upstream
		}
	}

	best.currentWeight -= b.totalWeight

	return best.url
}

// IPHash implements the IP Hashing strategy for selecting an upstream server.
type IPHash struct {
	upstreams []*url.URL
}

// NewIPHash returns the IPHash load balancer.
func NewIPHash(upstreams []*config.UpstreamServer) *IPHash {
	b := &IPHash{
		upstreams: make([]*url.URL, len(upstreams)),
	}

	for i, upstream := range upstreams {
		b.upstreams[i] = upstream.URL.URL
	}

	return b
}

// Select implements the IP Hashing algorithm to select and return
// the chosen upstream URL.
func (b *IPHash) Select(r *http.Request) *url.URL {
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr
	}

	return b.upstreams[xxhash.Sum64String(clientIP)%uint64(len(b.upstreams))]
}
