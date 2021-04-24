package sock

import "net"

type SockListener interface {
	OnSockOpen(c net.Conn)
	OnSockClose(c net.Conn)
	OnSockError(c net.Conn)
	OnHandlePack(c net.Conn, p *SockPack)
}
