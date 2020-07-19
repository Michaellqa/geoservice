package messaging

type Message struct {
	Body          interface{}
	CorrelationId string
	Sender        string
}

type Reply struct {
	Error error
	Body  interface{}
}
