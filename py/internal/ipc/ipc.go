package ipc

import "net"

// Listen returns a net.Listener appropriate for the current platform.
// addr: on Unix, a filesystem path (e.g. /tmp/insyra.sock)
// on Windows, a named pipe path (e.g. \\.\\pipe\\mypipe)
func Listen(addr string) (net.Listener, error) {
	return listen(addr)
}

// Dial connects to the given addr using the platform-appropriate transport.
func Dial(addr string) (net.Conn, error) {
	return dial(addr)
}
