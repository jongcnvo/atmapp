package rpc

import (
	"context"
	"net"
	"os"
	"path/filepath"
)

// CreateIPCListener creates an listener, on Unix platforms this is a unix socket, on
// Windows this is a named pipe
func CreateIPCListener(endpoint string) (net.Listener, error) {
	return ipcListen(endpoint)
}

// ipcListen will create a Unix socket on the given endpoint.
func ipcListen(endpoint string) (net.Listener, error) {
	// Ensure the IPC path exists and remove any previous leftover
	if err := os.MkdirAll(filepath.Dir(endpoint), 0751); err != nil {
		return nil, err
	}
	os.Remove(endpoint)
	l, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, err
	}
	os.Chmod(endpoint, 0600)
	return l, nil
}

// NewInProcClient attaches an in-process connection to the given RPC server.
func DialInProc(handler *Server) *Client {
	initctx := context.Background()
	c, _ := newClient(initctx, func(context.Context) (net.Conn, error) {
		p1, p2 := net.Pipe()
		go handler.ServeCodec(NewJSONCodec(p1), OptionMethodInvocation|OptionSubscriptions)
		return p2, nil
	})
	return c
}
