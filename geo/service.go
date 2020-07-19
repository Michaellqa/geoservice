package geo

import (
	"fmt"
	"geoservice/messaging"
	"github.com/rs/xid"
	"log"
	"time"
)

const pingInterval = 2 * time.Second

type Service interface {
	Start()
	Stop()
}

func NewService(b messaging.Broker) Service {
	s := &service{
		broker:  b,
		closing: make(chan chan struct{}, 2),
		loc:     RandomLocator(),
		name:    fmt.Sprintf("geo-%s", xid.New().String()),
	}
	return s
}

type service struct {
	broker  messaging.Broker
	closing chan chan struct{}
	loc     Locator
	name    string
}

func (s *service) Start() {
	go s.serve()
	go s.keepAlive()
	x, y := s.loc.Coordinates()
	log.Printf("%s [%f %f] is up\n", s.name, x, y)
}

func (s *service) serve() {
	reqCh := s.broker.Register(s.name)
	for {
		select {
		case msg, connected := <-reqCh:
			if !connected {
				reqCh = s.broker.Register(s.name)
				continue
			}
			s.process(msg)
		case ch := <-s.closing:
			ch <- struct{}{}
			break
		}
	}
}

func (s *service) keepAlive() {
	ticker := time.NewTicker(pingInterval)
	ping := messaging.Message{Sender: s.name}
	for {
		select {
		case <-ticker.C:
			s.broker.Send("", ping)
		case ch := <-s.closing:
			ch <- struct{}{}
			break
		}
	}
}

// Stop terminates processes of ping and accepting requests.
// Waits until they finish.
func (s *service) Stop() {
	ch := make(chan struct{})
	s.closing <- ch
	s.closing <- ch
	<-ch
	<-ch
}

func (s *service) process(msg messaging.Message) {
	// todo: see if it works
	r, ok := msg.Body.(Request)
	if !ok {
		return
	}

	switch r.Method {
	case MethodChangePosition:
		s.loc.Update()
		x, y := s.loc.Coordinates()
		log.Printf("%s updated location [%f %f]\n", s.name, x, y)
	case MethodGetDistance:
		dist := s.loc.Distance(r.X, r.Y)
		replyMsg := messaging.Message{
			Body:   dist,
			Sender: s.name,
		}
		s.broker.Response(msg.CorrelationId, replyMsg)
	}
}
