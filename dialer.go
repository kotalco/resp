package redis

import "net"

type IDialer interface {
	Dial(address string) (net.Conn, error)
}

type Dialer struct{}

func NewDialer() IDialer {
	return &Dialer{}
}
func (d Dialer) Dial(address string) (net.Conn, error) {
	return net.Dial("tcp", address)
}
