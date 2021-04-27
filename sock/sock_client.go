package sock

import (
	"net"
	"time"

	"github.com/wuyiyinxia/yxlib/util"
)

const (
	LOG_TAG_SC = "SockClient"
)

type SockClient struct {
	mgr *SockMgr
}

func NewSockClient(mgr *SockMgr) *SockClient {
	return &SockClient{
		mgr: mgr,
	}
}

func (c *SockClient) Connect(network string, address string, timeoutSec int64) (net.Conn, error) {
	conn, err := net.DialTimeout(network, address, time.Second*time.Duration(timeoutSec))
	if err != nil {
		util.Logger.E(LOG_TAG_SC, "dial error:", err)
		return nil, err
	}

	if c.mgr != nil {
		err = c.mgr.addConn(conn)
		if err != nil {
			util.Logger.E(LOG_TAG_SC, "add conn error:", err)
			return nil, err
		}
	}

	return conn, nil
}
