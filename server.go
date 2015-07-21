package weavebox

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Server provides a gracefull shutdown of http server.
type Server struct {
	*http.Server
	CloseTimeout time.Duration
	quit         chan struct{}
	wg           sync.WaitGroup
}

// ListenAndServe accepts http requests and start a goroutine for each request
func ListenAndServe(addr string, h http.Handler) error {
	s := &Server{
		Server:       &http.Server{Addr: addr, Handler: h},
		quit:         make(chan struct{}, 1),
		CloseTimeout: 400,
	}
	return s.listen()
}

// ListenAndServeTLS accepts http TLS encrypted requests and starts a goroutine
// for each request
func ListenAndServeTLS(addr string, h http.Handler, cert, key string) error {
	s := &Server{
		Server:       &http.Server{Addr: addr, Handler: h},
		quit:         make(chan struct{}, 1),
		CloseTimeout: 400,
	}
	return s.listenTLS(cert, key)
}

func (s *Server) listen() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.serve(l)
}

func (s *Server) listenTLS(cert, key string) error {
	var err error
	config := &tls.Config{}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}

	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	tlsList := tls.NewListener(l, config)
	return s.serve(tlsList)
}

// Serve hooks in the Server.ConnState to incr and decr the waitgroup based on
// the connection state.
func (s *Server) serve(l net.Listener) error {
	s.Server.ConnState = func(conn net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			s.wg.Add(1)
		case http.StateClosed, http.StateHijacked:
			s.wg.Done()
		}
	}
	go s.closeNotify(l)

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Server.Serve(l)
	}()

	for {
		select {
		case err := <-errChan:
			return err
		case <-s.quit:
			s.wg.Wait()
			break
		}
	}
	return nil
}

func (s *Server) closeNotify(l net.Listener) {
	sig := make(chan os.Signal, 1)

	signal.Notify(
		sig,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGQUIT,
		syscall.SIGUSR2,
		syscall.SIGINT,
	)
	sign := <-sig
	switch sign {
	case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
		l.Close()
		s.quit <- struct{}{}
	case syscall.SIGKILL:
		l.Close()
		s.quit <- struct{}{}
	case syscall.SIGUSR2:
		panic("USR2 => not implemented")
	}
}
