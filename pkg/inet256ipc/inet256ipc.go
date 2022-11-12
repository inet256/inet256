// package inet256ipc provides a generic client and server for using an inet256.Service across process boundaries.
package inet256ipc

import "time"

const (
	DefaultKeepAliveInterval = 1 * time.Second
	DefaultKeepAliveTimeout  = 3 * time.Second
)
