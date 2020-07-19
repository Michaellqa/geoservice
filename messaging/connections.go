package messaging

import (
	"log"
	"sync"
	"time"
)

type connections interface {
	Add(id string) peer
	Update(p peer)
	Client(id string) (peer, bool)
	Clients() []peer
}

type clients struct {
	peers map[string]peer
	mu    sync.RWMutex
}

func newClients() connections {
	cs := &clients{
		peers: make(map[string]peer),
		mu:    sync.RWMutex{},
	}
	go cs.startCleaner()
	return cs
}

func (c *clients) startCleaner() {
	t := time.Tick(time.Second)
	for range t {
		c.mu.Lock()
		expired := make([]string, 0)
		for _, p := range c.peers {
			if !p.isAlive() {
				close(p.closing)
				expired = append(expired, p.id)
			}
		}
		for _, id := range expired {
			delete(c.peers, id)
			log.Printf("[broker] connection to %s closed\n", id)
		}
		c.mu.Unlock()
	}
}

func (c *clients) Add(id string) peer {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := newPeer(id)
	c.peers[id] = p
	return p
}

func (c *clients) Update(p peer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.peers[p.id] = p
}

func (c *clients) Client(id string) (peer, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.peers[id]
	return p, ok
}

func (c *clients) Clients() []peer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ps := make([]peer, 0, len(c.peers))
	for _, p := range c.peers {
		ps = append(ps, p)
	}
	return ps
}
