//go:build windows

package collect

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"

	"github.com/ben/argus/internal/model"
)

var (
	iphlpapiDLL        = syscall.NewLazyDLL("iphlpapi.dll")
	icmpCreateFileProc = iphlpapiDLL.NewProc("IcmpCreateFile")
	icmpCloseHandleProc = iphlpapiDLL.NewProc("IcmpCloseHandle")
	icmpSendEchoProc   = iphlpapiDLL.NewProc("IcmpSendEcho")
)

const (
	ipSuccess       = 0
	ipReqTimedOut   = 11010
	defaultPingData = "argus"
)

type windowsProber struct{}

type ipOptionInformation struct {
	TTL         byte
	TOS         byte
	Flags       byte
	OptionsSize byte
	OptionsData uintptr
}

type icmpEchoReply struct {
	Address        uint32
	Status         uint32
	RoundTripTime  uint32
	DataSize       uint16
	Reserved       uint16
	Data           uintptr
	Options        ipOptionInformation
}

func NewProber() windowsProber {
	return windowsProber{}
}

func (windowsProber) Probe(ctx context.Context, target model.Target, timeout time.Duration) model.ProbeResult {
	select {
	case <-ctx.Done():
		return model.ProbeResult{Timestamp: time.Now(), Result: "error", Error: ctx.Err().Error()}
	default:
	}

	ip := net.ParseIP(target.IP)
	if ip == nil {
		return model.ProbeResult{Timestamp: time.Now(), Result: "error", Error: "target ip is invalid"}
	}
	ipv4 := ip.To4()
	if ipv4 == nil {
		return model.ProbeResult{Timestamp: time.Now(), Result: "error", Error: "only ipv4 targets are supported in v1"}
	}

	handle, _, callErr := icmpCreateFileProc.Call()
	if handle == 0 {
		return model.ProbeResult{Timestamp: time.Now(), Result: "error", Error: callErr.Error()}
	}
	defer icmpCloseHandleProc.Call(handle)

	payload := []byte(defaultPingData)
	replySize := int(unsafe.Sizeof(icmpEchoReply{})) + len(payload) + 8
	reply := make([]byte, replySize)
	ipAsUint32 := binary.BigEndian.Uint32(ipv4)

	ret, _, sendErr := icmpSendEchoProc.Call(
		handle,
		uintptr(ipAsUint32),
		uintptr(unsafe.Pointer(&payload[0])),
		uintptr(uint16(len(payload))),
		0,
		uintptr(unsafe.Pointer(&reply[0])),
		uintptr(uint32(replySize)),
		uintptr(uint32(timeout.Milliseconds())),
	)
	if ret == 0 {
		return model.ProbeResult{Timestamp: time.Now(), Result: "error", Error: sendErr.Error()}
	}

	parsed := (*icmpEchoReply)(unsafe.Pointer(&reply[0]))
	switch parsed.Status {
	case ipSuccess:
		rtt := int64(parsed.RoundTripTime)
		return model.ProbeResult{
			Timestamp: time.Now(),
			Result:    "success",
			RTTMs:     &rtt,
		}
	case ipReqTimedOut:
		return model.ProbeResult{
			Timestamp: time.Now(),
			Result:    "timeout",
			Error:     "request timeout",
		}
	default:
		return model.ProbeResult{
			Timestamp: time.Now(),
			Result:    "error",
			Error:     fmt.Sprintf("icmp status %d", parsed.Status),
		}
	}
}
