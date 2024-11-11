package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/hashicorp/yamux"
	"go.uber.org/zap"
)

// ListenAndProxy will listen for new client connections and start a stream over the established proxy server session.
// if the proxy server terminates the session, like when a grant expires, this listener will detect it and terminate the CLI commmand with an error explaining what happened
func ListenAndProxy(ctx context.Context, yamuxStreamConnection *yamux.Session, clientConnectionPort int, requestURL string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", clientConnectionPort))
	if err != nil {
		return fmt.Errorf("failed to start listening for connections on port: %d. %w", clientConnectionPort, err)
	}
	defer ln.Close()

	type result struct {
		conn net.Conn
		err  error
	}
	resultChan := make(chan result, 100)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := ln.Accept()
				result := result{
					err: err,
				}
				if err == nil {
					result.conn = conn
				}
				resultChan <- result
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-yamuxStreamConnection.CloseChan():
			return clierr.New("The connection to the proxy server has ended", clierr.Infof("Your grant may have expired, you can check the status here: %s and retry connecting", requestURL))
		case result := <-resultChan:
			if result.err != nil {
				return fmt.Errorf("failed to accept connection: %w", err)
			}
			if yamuxStreamConnection.IsClosed() {
				return clierr.New("failed to accept connection for client because the proxy server connection has ended", clierr.Infof("Your grant may have expired, you can check the status here: %s and retry connecting", requestURL))
			}
			go func(clientConn net.Conn) {

				// A stream is opened for this connection, streams are used just like a net.Conn and can read and write data
				// A stream can only be opened while the grant is still valid, and each new connection will validate the parameters
				sessionConn, err := yamuxStreamConnection.OpenStream()
				if err != nil {
					clio.Error("Failed to establish a new connection to the remote via the proxy server.")
					clio.Error(err)
					clio.Infof("Your grant may have expired, you can check the status here: %s", requestURL)
					return
				}

				clio.Infof("Connection accepted for session [%v]", sessionConn.StreamID())

				// If a stream successfully connects, that means that a connection to the target is now open
				// at this point the connection traffic is handed off and the connection is effectively directly from the client and the target
				// with queries being intercepted and logged to the audit trail in Common Fate
				// if the grant becomes incative at any time the connection is terminated immediately
				go func() {
					defer clientConn.Close()
					defer sessionConn.Close()
					_, err := io.Copy(sessionConn, clientConn)
					if err != nil {
						clio.Debugw("error writing data from client to server usually this is just because the proxy session ended.", "streamId", sessionConn.StreamID(), zap.Error(err))
					}
					clio.Infof("Connection ended for session [%v]", sessionConn.StreamID())
				}()
				go func() {
					defer clientConn.Close()
					defer sessionConn.Close()
					_, err := io.Copy(clientConn, sessionConn)
					if err != nil {
						clio.Debugw("error writing data from server to client usually this is just because the proxy session ended.", "streamId", sessionConn.StreamID(), zap.Error(err))
					}

				}()

				// This function polls the stream connection to see if it has been closed remotely
				// https://github.com/hashicorp/yamux/pull/115
				// when the proxy server has errors which are fatal to this session, it will close the stream
				go func() {
					defer clientConn.Close()
					defer sessionConn.Close()
					for {
						b := make([]byte, 0)
						_, err := sessionConn.Read(b)
						if err != nil {
							if errors.Is(err, io.EOF) {
								clio.Infof("The proxy server ended the connection for the session unexpectedly [%v]", sessionConn.StreamID())
								return
							}
						}
						time.Sleep(time.Second)
					}

				}()
			}(result.conn)
		}
	}
}
