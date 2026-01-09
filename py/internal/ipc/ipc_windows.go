//go:build windows
// +build windows

package ipc

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func listen(addr string) (net.Listener, error) {
	// addr should be like `\\.\\pipe\\mypipe`
	return winio.ListenPipe(addr, nil)
}

func dial(addr string) (net.Conn, error) {
	return winio.DialPipe(addr, nil)
}
