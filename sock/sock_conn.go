package sock

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

const (
	SOCK_READ_DEAD_LINE  = 8
	SOCK_WRITE_DEAD_LINE = 8
	SOCK_MAX_RESP_QUE    = 10
)

var (
	ErrStopSockWrite error = errors.New("stop write")
)

type SockConn struct {
	conn            net.Conn
	headerBuff      []byte
	requestQue      chan *SockPackWrap
	responeQue      chan *SockPack
	closeReadEvt    chan bool
	closeWriteEvt   chan bool
	exitEvt         chan bool
	headerProcessor SockHeaderProcessor
}

func NewSockConn(conn net.Conn, requestQue chan *SockPackWrap) *SockConn {
	c := &SockConn{
		conn:            conn,
		headerBuff:      make([]byte, SOCK_PACK_HEADER_LEN),
		requestQue:      requestQue,
		responeQue:      make(chan *SockPack, SOCK_MAX_RESP_QUE),
		closeReadEvt:    make(chan bool, 1),
		closeWriteEvt:   make(chan bool, 1),
		exitEvt:         make(chan bool, 1),
		headerProcessor: nil,
	}

	return c
}

func (c *SockConn) Start() {
	go c.read()
	go c.write()
}

func (c *SockConn) Stop() {
	if len(c.closeReadEvt) == 0 {
		fmt.Println("stop connect: ", c.conn.RemoteAddr())
		c.closeReadEvt <- true
	}
}

func (c *SockConn) SetHeaderProcessor(headerProcessor SockHeaderProcessor) {
	c.headerProcessor = headerProcessor
}

func (c *SockConn) CanRemove() bool {
	retCode := false

	select {
	case <-c.exitEvt:
		retCode = true
	default:
	}

	return retCode
}

func (c *SockConn) PushRespone(p *SockPack) {
	c.responeQue <- p
}

//===============================
//           read
//===============================
func (c *SockConn) read() {
	for {
		if c.isCloseRead() {
			break
		}

		// read header
		var len uint16 = 0
		var err error = nil
		if c.headerProcessor != nil {
			len, err = c.headerProcessor.ReadHeader(c.headerBuff)
		} else {
			len, err = c.readHeader(c.headerBuff)
		}

		if err != nil {
			fmt.Println("read data len error: ", err)
			break
		}

		buff := make([]byte, len+SOCK_PACK_HEADER_LEN)
		copy(buff, c.headerBuff)

		// read data
		if len > 0 {
			err = c.readData(buff[SOCK_PACK_HEADER_LEN:])
			if err != nil {
				fmt.Println("read data error: ", err)
				break
			}
		}

		// unpack
		p, err := c.unpack(buff)
		if err != nil {
			fmt.Println("unpack error: ", err)
			break
		}

		// push to request queue
		c.requestQue <- NewSockPackWrap(p, c.conn)
	}

	c.closeWriteEvt <- true
}

func (c *SockConn) isCloseRead() bool {
	retCode := false

	select {
	case <-c.closeReadEvt:
		retCode = true
	default:
	}

	return retCode
}

func (c *SockConn) readHeader(buff []byte) (uint16, error) {
	buffLen := len(buff)
	_, err := c.readToBuff(buff)
	if err != nil {
		return 0, err
	}

	// check package mark
	var mark uint16 = 0
	markBuff := buff[:SOCK_PACK_MARK_LEN]
	buffWrap := bytes.NewBuffer(markBuff)
	err = binary.Read(buffWrap, binary.BigEndian, &mark)
	if err != nil {
		return 0, err
	}

	if mark != GetPackMark() {
		err = errors.New("wrong start mark")
		return 0, err
	}

	// get data len
	var dataLen uint16 = 0
	dataLenBuff := buff[buffLen-2:]
	buffWrap = bytes.NewBuffer(dataLenBuff)
	err = binary.Read(buffWrap, binary.BigEndian, &dataLen)
	if err != nil {
		return 0, err
	}

	return dataLen, nil
}

func (c *SockConn) readData(buff []byte) error {
	_, err := c.readToBuff(buff)
	return err
}

func (c *SockConn) readToBuff(buff []byte) (int, error) {
	var err error = nil
	totalSize := 0
	buffLen := len(buff)

	for {
		if c.isCloseRead() {
			err = errors.New("stop read")
			break
		}

		err = c.conn.SetReadDeadline(time.Now().Add(time.Second * SOCK_READ_DEAD_LINE))
		if err != nil {
			break
		}

		n, err := c.conn.Read(buff[totalSize:])
		if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
			totalSize += n
		}

		if errors.Is(err, os.ErrDeadlineExceeded) && totalSize < buffLen {
			continue
		}

		break
	}

	if err == nil && totalSize < buffLen {
		err = errors.New("unexpect read end")
	}

	return totalSize, err
}

func (c *SockConn) unpack(buff []byte) (*SockPack, error) {
	p := NewSockPack()
	headerBuff := buff[SOCK_PACK_MARK_LEN:SOCK_PACK_HEADER_LEN]
	buffWrap := bytes.NewBuffer(headerBuff)

	// cmd
	err := binary.Read(buffWrap, binary.BigEndian, &p.Cmd)
	if err != nil {
		return nil, err
	}

	// src
	err = binary.Read(buffWrap, binary.BigEndian, &p.SrcEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffWrap, binary.BigEndian, &p.SrcNo)
	if err != nil {
		return nil, err
	}

	// dst
	err = binary.Read(buffWrap, binary.BigEndian, &p.DstEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffWrap, binary.BigEndian, &p.DstNo)
	if err != nil {
		return nil, err
	}

	// data len
	err = binary.Read(buffWrap, binary.BigEndian, &p.DataLen)
	if err != nil {
		return nil, err
	}

	// data
	if p.DataLen > 0 {
		p.Data = buff[SOCK_PACK_HEADER_LEN:]
	}

	p.RawBuff = buff

	return p, nil
}

//===============================
//           write
//===============================
func (c *SockConn) write() {
	var err error = nil
	isExit := false

	// loop
	for {
		isExit, err = c.writeLogic()
		if isExit {
			break
		}
	}

	// handle error
	/*if err == ErrStopSockWrite {

	} else */if err != nil {
		fmt.Println("write pack error: ", err)
		c.Stop()
		c.waitCloseWrite()
	} else {
		c.writeAllResp()
	}

	// close
	err = c.conn.Close()
	if err != nil {
		fmt.Println("close conn failed!")
	}

	// notify exit
	c.exitEvt <- true
}

func (c *SockConn) writeLogic() (bool, error) {
	var err error = nil
	isExit := false

	select {
	case p := <-c.responeQue:
		err = c.writePack(p)
		if err != nil {
			isExit = true
		}

	case <-c.closeWriteEvt:
		isExit = true
	}

	return isExit, err
}

func (c *SockConn) writePack(p *SockPack) error {
	buff, err := c.pack(p)
	if err != nil {
		return err
	}

	if c.headerProcessor != nil {
		err = c.headerProcessor.WriteHeader(buff[:SOCK_PACK_HEADER_LEN])
	} else {
		err = c.writeHeader(buff[:SOCK_PACK_HEADER_LEN])
	}

	if err != nil {
		return err
	}

	err = c.writeData(buff[SOCK_PACK_HEADER_LEN:])
	if err != nil {
		return err
	}

	return nil
}

func (c *SockConn) writeHeader(buff []byte) error {
	return c.writeBuff(buff)
}

func (c *SockConn) writeData(buff []byte) error {
	return c.writeBuff(buff)
}

func (c *SockConn) writeBuff(buff []byte) error {
	buffLen := len(buff)
	totalSize, err := c.conn.Write(buff)
	if err == nil && totalSize < buffLen {
		err = errors.New("unexpect write end")
	}

	return err
}

func (c *SockConn) pack(p *SockPack) ([]byte, error) {
	if p.RawBuff != nil {
		return p.RawBuff, nil
	}

	var dataLen uint16 = 0
	if p.Data != nil {
		dataLen = uint16(len(p.Data))
	}

	buff := make([]byte, SOCK_PACK_HEADER_LEN+dataLen)
	buffWrap := bytes.NewBuffer(buff)

	// mark
	var mark uint16 = GetPackMark()
	err := binary.Write(buffWrap, binary.BigEndian, &mark)
	if err != nil {
		return nil, err
	}

	// cmd
	err = binary.Write(buffWrap, binary.BigEndian, &p.Cmd)
	if err != nil {
		return nil, err
	}

	// src
	var tmpEnd uint8 = uint8(p.SrcEnd)
	err = binary.Write(buffWrap, binary.BigEndian, &tmpEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffWrap, binary.BigEndian, &p.SrcNo)
	if err != nil {
		return nil, err
	}

	// dst
	tmpEnd = uint8(p.DstEnd)
	err = binary.Write(buffWrap, binary.BigEndian, &tmpEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffWrap, binary.BigEndian, &p.DstNo)
	if err != nil {
		return nil, err
	}

	// data len
	err = binary.Write(buffWrap, binary.BigEndian, &dataLen)
	if err != nil {
		return nil, err
	}

	// data
	if dataLen > 0 {
		subBuff := buff[SOCK_PACK_HEADER_LEN:]
		copy(subBuff, p.Data)
	}

	return buff, nil
}

func (c *SockConn) waitCloseWrite() {
	<-c.closeWriteEvt
}

func (c *SockConn) writeAllResp() {
	for {
		select {
		case p := <-c.responeQue:
			c.writePack(p)
		default:
			goto Exit0
		}
	}

Exit0:
	return
}
