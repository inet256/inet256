package inet256mem

type config struct {
	queueLen int
}

type Option func(*config)

// WithQueueLen sets the receive queue len in messages for Nodes.
func WithQueueLen(l int) Option {
	return func(c *config) {
		c.queueLen = l
	}
}
