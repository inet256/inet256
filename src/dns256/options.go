package dns256

type ResolverOpt func(c *Resolver)

type resolveConfig struct {
	maxHops int
}

type ResolveOpt func(c *resolveConfig)

func WithMaxHops(n int) ResolveOpt {
	return func(c *resolveConfig) {
		c.maxHops = n
	}
}
