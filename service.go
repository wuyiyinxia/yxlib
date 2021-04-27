package yxlib

import (
	"net"

	"github.com/wuyiyinxia/yxlib/sock"
)

type Service interface {
	OnHandlePack(p *sock.SockPack, c net.Conn) error
}
