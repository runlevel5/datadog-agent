// Code generated by cmd/cgo -godefs; DO NOT EDIT.
// cgo -godefs -- -I ../../ebpf/c -I ../../../ebpf/c -fsigned-char types.go

package postgres

type ConnTuple = struct {
	Saddr_h  uint64
	Saddr_l  uint64
	Daddr_h  uint64
	Daddr_l  uint64
	Sport    uint16
	Dport    uint16
	Netns    uint32
	Pid      uint32
	Metadata uint32
}

type EbpfEvent struct {
	Tuple ConnTuple
	Tx    EbpfTx
}
type EbpfTx struct {
	Request_fragment   [64]byte
	Request_started    uint64
	Response_last_seen uint64
	Frag_size          uint8
	Pad_cgo_0          [7]byte
}

const (
	BufferSize = 0x40
)
