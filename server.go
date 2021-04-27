package yxlib

import (
	"errors"
	"net"

	"github.com/wuyiyinxia/yxlib/sock"
)

type Server struct {
	mgr         *sock.SockMgr
	serv        *sock.SockServ
	client      *sock.SockClient
	mapMod2Serv map[uint16]Service
}

var CurServ *Server = nil

func NewServer(endType uint8, endNo uint16) *Server {
	s := &Server{
		mgr:         nil,
		serv:        nil,
		client:      nil,
		mapMod2Serv: make(map[uint16]Service),
	}

	s.mgr = sock.NewSockMgr(endType, endNo)
	s.serv = sock.NewSockServ(s.mgr)
	s.client = sock.NewSockClient(s.mgr)
	CurServ = s
	return s
}

func (s *Server) AddService(mod uint16, serv Service) error {
	if serv == nil {
		return errors.New("service is nil")
	}

	s.mapMod2Serv[mod] = serv
	return nil
}

func (s *Server) SetSockListener(l sock.SockListener) {
	s.mgr.SetListener(l)
}

func (s *Server) Start(network string, address string) error {
	err := s.serv.Listen(network, address)
	if err != nil {
		return err
	}

	go s.serv.Start()
	s.mgr.Start()
	return nil
}

func (s *Server) Connect(network string, address string, timeoutSec int64) (net.Conn, error) {
	return s.client.Connect(network, address, timeoutSec)
}

func (s *Server) CloseConn(c net.Conn) {
	s.mgr.CloseConn(c)
}

func (s *Server) SetHeaderProcessor(headerProcessor sock.SockHeaderProcessor, c net.Conn) {
	s.mgr.SetHeaderProcessor(headerProcessor, c)
}

func (s *Server) HandlePack(p *sock.SockPack, c net.Conn, mod uint16) error {
	serv := s.mapMod2Serv[mod]
	if nil == serv {
		return errors.New("no service for this mod")
	}

	return serv.OnHandlePack(p, c)
}

func (s *Server) Send(p *sock.SockPack, c net.Conn) {
	s.mgr.Send(p, c)
}

func (s *Server) Stop() {
	s.serv.Stop()
	s.mgr.Stop()
}
