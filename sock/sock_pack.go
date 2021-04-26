package sock

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
)

const (
	SOCK_PACK_MARK_LEN   = 2
	SOCK_PACK_HEADER_LEN = 12
)

var sockPackMark uint16 = 0x5958

func SetPackMark(mark uint16) {
	sockPackMark = mark
}

func GetPackMark() uint16 {
	return sockPackMark
}

/*
 * @struct SockPackWrap
 * Serialized data:
 * 2 bytes for mark, 2 bytes for command,
 * 1 byte for source type, 2 byte for source number,
 * 1 byte for dest type, 2 byte for dest number,
 * 2 byte for data length,
 * the rest is data
 */
type SockPack struct {
	Cmd     uint16
	SrcEnd  uint8
	SrcNo   uint16
	DstEnd  uint8
	DstNo   uint16
	DataLen uint16
	Data    []byte
	RawBuff []byte // whold package stream data
}

func NewSockPack() *SockPack {
	return &SockPack{
		Cmd:     0,
		SrcEnd:  0,
		SrcNo:   0,
		DstEnd:  0,
		DstNo:   0,
		DataLen: 0,
		Data:    nil,
		RawBuff: nil,
	}
}

func NewReqSockPack(cmd uint16, srcEnd uint8, srcNo uint16, dstEnd uint8, dstNo uint16) *SockPack {
	return &SockPack{
		Cmd:     cmd,
		SrcEnd:  srcEnd,
		SrcNo:   srcNo,
		DstEnd:  dstEnd,
		DstNo:   dstNo,
		DataLen: 0,
		Data:    nil,
		RawBuff: nil,
	}
}

func GetRespSockPack(p *SockPack) *SockPack {
	return &SockPack{
		Cmd:     p.Cmd,
		SrcEnd:  p.DstEnd,
		SrcNo:   p.DstNo,
		DstEnd:  p.SrcEnd,
		DstNo:   p.SrcNo,
		DataLen: 0,
		Data:    nil,
		RawBuff: nil,
	}
}

func (p *SockPack) GetDataFromRaw() []byte {
	if p.RawBuff == nil || len(p.RawBuff) <= SOCK_PACK_HEADER_LEN {
		return nil
	}

	if p.DataLen == 0 {
		return nil
	}

	return p.RawBuff[SOCK_PACK_HEADER_LEN:]
}

/*
 * @struct SockPackWrap
 */
type SockPackWrap struct {
	Pack *SockPack
	Conn net.Conn
}

func NewSockPackWrap(pack *SockPack, c net.Conn) *SockPackWrap {
	wrap := &SockPackWrap{
		Pack: pack,
		Conn: c,
	}

	return wrap
}

/*
 * @interface SockPakcer
 */
type SockPakcer interface {
	ReadHeader(buff []byte, c *SockConn) (uint16, error)
	Unpack(buff []byte, c *SockConn) (*SockPack, error)
	Pack(pack *SockPack, c *SockConn) ([]byte, error)
}

/*
 * @struct DefSockPacker
 */
type DefSockPacker struct {
}

func NewDefSockPacker() *DefSockPacker {
	return &DefSockPacker{}
}

func (p *DefSockPacker) ReadHeader(buff []byte, c *SockConn) (uint16, error) {
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

func (p *DefSockPacker) Unpack(buff []byte, c *SockConn) (*SockPack, error) {
	pack := NewSockPack()
	headerBuff := buff[SOCK_PACK_MARK_LEN:SOCK_PACK_HEADER_LEN]
	buffWrap := bytes.NewBuffer(headerBuff)

	// cmd
	err := binary.Read(buffWrap, binary.BigEndian, &pack.Cmd)
	if err != nil {
		return nil, err
	}

	// src
	err = binary.Read(buffWrap, binary.BigEndian, &pack.SrcEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffWrap, binary.BigEndian, &pack.SrcNo)
	if err != nil {
		return nil, err
	}

	// dst
	err = binary.Read(buffWrap, binary.BigEndian, &pack.DstEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffWrap, binary.BigEndian, &pack.DstNo)
	if err != nil {
		return nil, err
	}

	// data len
	err = binary.Read(buffWrap, binary.BigEndian, &pack.DataLen)
	if err != nil {
		return nil, err
	}

	// data
	if pack.DataLen > 0 {
		pack.Data = buff[SOCK_PACK_HEADER_LEN:]
	}

	pack.RawBuff = buff

	return pack, nil
}

func (p *DefSockPacker) Pack(pack *SockPack, c *SockConn) ([]byte, error) {
	if pack.RawBuff != nil {
		return pack.RawBuff, nil
	}

	var dataLen uint16 = 0
	if pack.Data != nil {
		dataLen = uint16(len(pack.Data))
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
	err = binary.Write(buffWrap, binary.BigEndian, &pack.Cmd)
	if err != nil {
		return nil, err
	}

	// src
	var tmpEnd uint8 = uint8(pack.SrcEnd)
	err = binary.Write(buffWrap, binary.BigEndian, &tmpEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffWrap, binary.BigEndian, &pack.SrcNo)
	if err != nil {
		return nil, err
	}

	// dst
	tmpEnd = uint8(pack.DstEnd)
	err = binary.Write(buffWrap, binary.BigEndian, &tmpEnd)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffWrap, binary.BigEndian, &pack.DstNo)
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
		copy(subBuff, pack.Data)
	}

	return buff, nil
}
