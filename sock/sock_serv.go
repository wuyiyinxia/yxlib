package sock

import (
	"fmt"
	"net"
)

type SockServ struct {
	l        net.Listener
	mgr      *SockMgr
	closeEvt chan bool
}

func NewSockServ(mgr *SockMgr) *SockServ {
	return &SockServ{
		l:        nil,
		mgr:      mgr,
		closeEvt: make(chan bool, 1),
	}
}

func (s *SockServ) Listen(network string, address string) error {
	l, err := net.Listen(network, address)
	if err != nil {
		return err
	}

	s.l = l
	return nil
}

func (s *SockServ) Start() {
	fmt.Println("server start")

	for {
		select {
		case <-s.closeEvt:
			goto Exit0

		default:
			err := s.accept()
			if err != nil {
				fmt.Println("accept error:", err)
				<-s.closeEvt
				goto Exit0
			}
		}
	}

Exit0:
	fmt.Println("server stop")
}

func (s *SockServ) Stop() {
	s.closeEvt <- true
	s.l.Close()
}

func (s *SockServ) accept() error {
	c, err := s.l.Accept()
	if err != nil {
		return err
	}

	if s.mgr != nil {
		err = s.mgr.AddConn(c)
		if err != nil {
			return err
		}
	}

	return nil
}
