package weavebox

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bradfitz/http2"
)

const useClosedConn = "use of closed network connection"

// Server provides a gracefull shutdown of http server.
type server struct {
	*http.Server
	quit  chan struct{}
	fquit chan struct{}
	wg    sync.WaitGroup
}

func newServer(addr string, h http.Handler, HTTP2 bool) *server {
	srv := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if HTTP2 {
		http2.ConfigureServer(srv, &http2.Server{})
	}
	return &server{
		Server: srv,
		quit:   make(chan struct{}, 1),
		fquit:  make(chan struct{}, 1),
	}
}

// ListenAndServe accepts http requests and start a goroutine for each request
func ListenAndServe(addr string, h http.Handler, HTTP2 bool) error {
	s := newServer(addr, h, HTTP2)
	return s.listen()
}

// ListenAndServeTLS accepts http TLS encrypted requests and starts a goroutine
// for each request
func ListenAndServeTLS(addr string, h http.Handler, cert, key string) error {
	s := newServer(addr, h, true)
	return s.listenTLS(cert, key)
}

func (s *server) listen() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.serve(l)
}

func (s *server) listenTLS(cert, key string) error {
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

// serve hooks in the Server.ConnState to incr and decr the waitgroup based on
// the connection state.
func (s *server) serve(l net.Listener) error {
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
			if strings.Contains(err.Error(), useClosedConn) {
				continue
			}
			return err
		case <-s.quit:
			s.SetKeepAlivesEnabled(false)
			s.wg.Wait()
			return errors.New("server stopped gracefully")
		case <-s.fquit:
			return errors.New("server stopped: process killed")
		}
	}
}

func (s *server) closeNotify(l net.Listener) {
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
		s.fquit <- struct{}{}
	case syscall.SIGUSR2:
		panic("USR2 => not implemented")
	}
}
