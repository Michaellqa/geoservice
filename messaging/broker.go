package messaging

import (
	"errors"
	"github.com/rs/xid"
	"log"
	"regexp"
	"sync"
	"time"
)

var (
	NotFound       = errors.New("not found")
	TimedOut       = errors.New("timeout")
	defaultTimeout = 5 * time.Second
)

type Broker interface {
	Register(name string) chan Message          // Регистрирует новый сервис в системе, возвращает канал по которому будут приходить запросы
	Broadcast(routingKey string, msg Message)   // Отправляет сообщение всем зарегистрированным сервисам
	Send(target string, msg Message) chan Reply // Отправляет сообщение target сервису и возвращает канал в который будет отправлен ответ
	Response(target string, msg Message)        // Отправляет ответ на указанную ноду
	GetServices() []string                      // Возвращает имена всех зарегистрированных сервисов
}

func NewBroker() Broker {
	return &broker{
		cons: newClients(),
		mu:   sync.RWMutex{},
	}
}

type broker struct {
	cons    connections
	mu      sync.RWMutex
	waiting map[string]chan Reply
}

// Register
func (b *broker) Register(name string) chan Message {
	c := b.cons.Add(name)
	return c.dataChan
}

// Broadcast accepts routingKey parameter as a regexp pattern. The message
// will be sent only to the services which name matches the key. If Empty
// has passed then message will be sent to all services.
func (b *broker) Broadcast(routingKey string, msg Message) {
	if routingKey == "" {
		routingKey = ".*"
	}
	exp, err := regexp.Compile(routingKey)
	if err != nil {
		log.Println(err)
		return
	}
	clients := b.cons.Clients()
	for _, cl := range clients {
		if !exp.MatchString(cl.id) {
			continue
		}
		go func() {
			timer := time.NewTimer(defaultTimeout)
			select {
			case <-timer.C:
			case cl.dataChan <- msg:
				timer.Stop()
			}
		}()
	}
}

// Send looks up for the targeted service and sends msg into its dataChan.
// If it cannot find one the error message is sent to the dataChan.
// After sending message waits for the response from the same dataChan and
// transfers writes it to the response dataChan.
func (b *broker) Send(target string, msg Message) chan Reply {
	replyCh := make(chan Reply, 1)
	// handle ping requests
	if target == "" {
		cl, ok := b.cons.Client(msg.Sender)
		if ok {
			replyCh <- Reply{Error: NotFound}
			return replyCh
		}
		cl.wake()
		b.cons.Update(cl) // todo: improve connections api
		replyCh <- Reply{}
		return replyCh
	}
	// initiate async request with correlation id
	cl, ok := b.cons.Client(target)
	if !ok {
		replyCh <- Reply{Error: NotFound}
		return replyCh
	}
	go func() {
		msg.CorrelationId = xid.New().String()
		timer := time.NewTimer(defaultTimeout)
		select {
		case <-timer.C:
			replyCh <- Reply{Error: TimedOut}
		case cl.dataChan <- msg:
			timer.Stop()
			b.mu.Lock()
			b.waiting[msg.CorrelationId] = replyCh
			b.mu.Unlock()
		}
	}()
	return replyCh
}

func (b *broker) Response(target string, msg Message) {
	b.mu.RLock()
	replyCh, ok := b.waiting[target]
	b.mu.Unlock()
	if !ok {
		return
	}
	go func() {
		reply := Reply{Body: msg.Body}
		timer := time.NewTimer(defaultTimeout)
		select {
		case <-timer.C:
			log.Printf("timeout on sending to %s", target)
		case replyCh <- reply:
			timer.Stop()
			b.mu.Lock()
			delete(b.waiting, target)
			b.mu.Unlock()
		}
	}()
}

// GetServices returns names of all the registered and active services.
func (b *broker) GetServices() []string {
	clients := b.cons.Clients()
	names := make([]string, 0, len(clients))
	for _, c := range clients {
		names = append(names, c.id)
	}
	return names
}
