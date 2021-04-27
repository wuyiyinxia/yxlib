package sock

import (
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

type SockHeaderProcessor interface {
	ReadHeader(buff []byte) (uint16, error)
	WriteHeader(buff []byte) error
}
