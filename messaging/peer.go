package messaging

import "time"

const (
	ttl = 4 * time.Second
)

// peer holds client's connection metadata.
type peer struct {
	id       string
	dataChan chan Message
	closing  chan struct{}
	expired  time.Time
}

func newPeer(id string) peer {
	return peer{
		id:       id,
		dataChan: make(chan Message),
		closing:  make(chan struct{}, 0),
		expired:  time.Now().Add(ttl),
	}
}

// Wake updates the expiration time
func (p *peer) wake() {
	p.expired = time.Now().Add(ttl)
}

func (p *peer) isAlive() bool {
	return p.expired.After(time.Now())
}
