package main

import (
	"context"
	"net"
)

type SocketClient struct {
	Service  *Service
	Conn     net.Conn
	Listener *SocketClientListener
	Hub      *SocketHub
	Message  chan []byte
	Signal   chan string
	Context  context.Context
	Cancel   context.CancelFunc
}

func (s *SocketClient) HandConn(conn net.Conn, bytes int) {

	defer func() {
		s.Conn.Close()
		if s.Clean() {
			s.Listener.ClientRecycler <- s
		}
		p := recover()
		CheckPanic(p, s.Service, "Client HandConn panic")
	}()

	s.Conn = conn
	s.Hub.Register <- s
	//s.Conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	//SocketRead(s.Conn, s.Hub.Receiver, s.Service)
Circle:
	for {
		var data = make([]byte, bytes, bytes)
		//_, err := s.Conn.Read(data)
		//CheckErr(err)
		if !SecureRead(data, conn, s.Service) {
			s.Service.Info <- "Socket Client Read Error. Addr: " + s.Conn.RemoteAddr().String()
			s.Cancel()
			break
		}

		select {
		case <-s.Context.Done():
			s.Service.Info <- "Socket Client HandConn Done. Addr: " + s.Conn.RemoteAddr().String()
			break Circle
		default:
		}

		s.Hub.Receiver <- data
	}
}

func (s *SocketClient) Broadcast() {
Circle:
	for {
		select {
		case <-s.Context.Done():
			s.Service.Info <- "Socket Client Broadcast Done. Addr: " + s.Conn.RemoteAddr().String()
			break Circle
		case message := <-s.Message:
			if !SecureWrite(message, s.Conn, s.Service) {
				s.Cancel()
			}
		}
	}
}

func (s *SocketClient) Clean() (ok bool) {
	defer func() {
		p := recover()
		if !CheckPanic(p, s.Service, "Hub Clean panic!") {
			ok = false
		}
	}()

	close(s.Message)
	close(s.Signal)
	s.Message = make(chan []byte, 100)
	s.Signal = make(chan string, 5)

	ok = true
	return
}

func NewSocketClient() *SocketClient {
	return &SocketClient{
		Message: make(chan []byte, 100),
		Signal:  make(chan string, 5),
		//Context: context.Background(),
		//Cancel:  func() {},
	}
}
