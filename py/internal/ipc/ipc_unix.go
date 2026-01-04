//go:build !windows
// +build !windows

package ipc

import (
	"net"
)

func listen(addr string) (net.Listener, error) {
	return net.Listen("unix", addr)
}

func dial(addr string) (net.Conn, error) {
	return net.Dial("unix", addr)
}
