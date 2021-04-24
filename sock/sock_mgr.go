package sock

import (
	"errors"
	"net"
	"time"
)

const (
	SOCK_CONN_ADD_QUE_MAX     uint16        = 1024
	SOCK_CONN_CLOSE_QUE_MAX   uint16        = 1024
	SOCK_SEND_QUE_MAX         uint16        = 1024
	SOCK_RECV_QUE_MAX         uint16        = 1024
	SOCK_MAINTAIN_INTV        time.Duration = (2 * time.Minute)
	SOCK_MGR_CLOSE_DELAY      time.Duration = (2 * time.Minute)
	SOCK_CHECK_ALL_CLOSE_INTV time.Duration = (2 * time.Second)
)

type SockMgr struct {
	endType      uint8
	endNo        uint16
	mapConn      map[net.Conn]*SockConn
	connAddQue   chan net.Conn
	connCloseQue chan net.Conn
	sendQue      chan *SockPackWrap
	recvQue      chan *SockPackWrap
	closeEvt     chan bool
	stopAddEvt   chan bool
	listener     SockListener
}

func NewSockMgr(endType uint8, endNo uint16) *SockMgr {
	return &SockMgr{
		endType:      endType,
		endNo:        endNo,
		mapConn:      make(map[net.Conn]*SockConn),
		connAddQue:   make(chan net.Conn, SOCK_CONN_ADD_QUE_MAX),
		connCloseQue: make(chan net.Conn, SOCK_CONN_CLOSE_QUE_MAX),
		sendQue:      make(chan *SockPackWrap, SOCK_SEND_QUE_MAX),
		recvQue:      make(chan *SockPackWrap, SOCK_RECV_QUE_MAX),
		closeEvt:     make(chan bool, 1),
		stopAddEvt:   make(chan bool, 1),
		listener:     nil,
	}
}

func (m *SockMgr) SetListener(l SockListener) {
	m.listener = l
}

func (m *SockMgr) AddConn(c net.Conn) error {
	var err error = nil
	if len(m.stopAddEvt) == 0 {
		m.connAddQue <- c
	} else {
		err = errors.New("stop add conn")
	}

	return err
}

func (m *SockMgr) CloseConn(c net.Conn) {
	m.connCloseQue <- c
}

func (m *SockMgr) Send(p *SockPack, c net.Conn) {
	wrap := NewSockPackWrap(p, c)
	m.sendQue <- wrap
}

func (m *SockMgr) Start() {
	ticker := time.NewTicker(SOCK_MAINTAIN_INTV)

	for {
		select {
		case c := <-m.connAddQue:
			m.handleAddConn(c)

		case c := <-m.connCloseQue:
			m.handleCloseConn(c)

		case wrap := <-m.sendQue:
			m.handleSend(wrap)

		case wrap := <-m.recvQue:
			m.handleRecv(wrap)

		case <-ticker.C:
			m.handleTicker()

		case <-m.closeEvt:
			goto Exit0
		}
	}

Exit0:
	ticker.Stop()
	m.handleExit()
}

func (m *SockMgr) Stop() {
	m.stopAddEvt <- true
	m.closeEvt <- true
}

func (m *SockMgr) handleAddConn(c net.Conn) {
	conn := NewSockConn(c, m.recvQue)
	m.mapConn[c] = conn

	if m.listener != nil {
		m.listener.OnSockOpen(c)
	}

	conn.Start()
}

func (m *SockMgr) handleCloseConn(c net.Conn) {
	conn := m.mapConn[c]
	if conn == nil {
		return
	}

	conn.Stop()
}

func (m *SockMgr) handleSend(wrap *SockPackWrap) {
	c := wrap.Conn
	p := wrap.Pack
	conn := m.mapConn[c]
	if conn == nil {
		return
	}

	conn.PushRespone(p)
}

func (m *SockMgr) handleRecv(wrap *SockPackWrap) {
	if m.listener != nil {
		m.listener.OnHandlePack(wrap.Conn, wrap.Pack)
	}
}

func (m *SockMgr) handleTicker() {
	var removeKeys []net.Conn = make([]net.Conn, 0)
	for k, v := range m.mapConn {
		if v.CanRemove() {
			removeKeys = append(removeKeys, k)
		}
	}

	for _, key := range removeKeys {
		if m.listener != nil {
			m.listener.OnSockClose(key)
		}

		delete(m.mapConn, key)
	}
}

func (m *SockMgr) handleExit() {
	for _, conn := range m.mapConn {
		conn.Stop()
	}

	ticker := time.NewTicker(SOCK_CHECK_ALL_CLOSE_INTV)
	for {
		if m.waitCloseAllConn(ticker) {
			break
		}
	}

	ticker.Stop()
}

func (m *SockMgr) waitCloseAllConn(ticker *time.Ticker) bool {
	bRetCode := false

	<-ticker.C
	m.handleTicker()
	if len(m.mapConn) == 0 {
		bRetCode = true
	}
	// select {
	// case <-ticker.C:
	// 	m.handleTicker()
	// 	if len(m.mapConn) == 0 {
	// 		bRetCode = true
	// 	}
	// }

	return bRetCode
}
