package connpool

import (
	log "github.com/cihub/seelog"
	"sync"
	"syscall"
)

var mu = sync.RWMutex{}

type ConnPool struct {
	fds       chan int
	sockType  int
	sockProto int
}

var globalConnPoolRaw *ConnPool
var globalConnPoolDgram *ConnPool

func GetGlobalConnPollRaw() *ConnPool {
	mu.Lock()
	defer mu.Unlock()
	if globalConnPoolRaw == nil {
		globalConnPoolRaw = NewConnPool(syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
		return globalConnPoolRaw
	}
	return globalConnPoolRaw
}

func GetGlobalConnPollDgram() *ConnPool {
	mu.Lock()
	defer mu.Unlock()
	if globalConnPoolDgram == nil {
		globalConnPoolDgram = NewConnPool(syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
		return globalConnPoolDgram
	}
	return globalConnPoolDgram
}

func NewConnPool(sockType int, proto int) *ConnPool {
	return &ConnPool{
		fds:       make(chan int, 100),
		sockType:  sockType,
		sockProto: proto,
	}
}

func (c *ConnPool) Get() (int, error) {
	log.Info("[Traceroute] Get1")
	if len(c.fds) == 0 {
		log.Info("[Traceroute] Get2")
		// Set up the socket to receive inbound packets
		socketFd, err := syscall.Socket(syscall.AF_INET, c.sockType, c.sockProto)
		log.Info("[Traceroute] Get3")
		if err != nil {
			log.Warnf("[Connection Pool] error: %s", err)
			return 0, err
		}
		return socketFd, nil
	}
	log.Info("[Traceroute] Get4")
	return <-c.fds, nil
}

func (c *ConnPool) Release(fd int) {
	log.Infof("[Traceroute] Release: %d", fd)
	c.fds <- fd
}

func (c *ConnPool) Size() int {
	return len(c.fds)
}

func (c *ConnPool) Cap() int {
	return cap(c.fds)
}
