// Package netstat measures throughput of a network interface from sysfs counters.
package netstat

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Sampler turns the kernel's cumulative byte counters into rates.
type Sampler struct {
	iface  string
	rx, tx uint64
	at     time.Time
}

func NewSampler(iface string) *Sampler {
	return &Sampler{iface: iface}
}

// Sample returns bytes per second since the previous call. ok is false when
// the interface is missing or there is no previous sample to compare against.
func (s *Sampler) Sample() (rx, tx float64, ok bool) {
	curRx, errRx := s.read("rx_bytes")
	curTx, errTx := s.read("tx_bytes")
	now := time.Now()
	if errRx != nil || errTx != nil {
		s.at = time.Time{}
		return 0, 0, false
	}
	prevRx, prevTx, prevAt := s.rx, s.tx, s.at
	s.rx, s.tx, s.at = curRx, curTx, now
	// Counters restart when the interface is recreated on reconnect.
	if prevAt.IsZero() || curRx < prevRx || curTx < prevTx {
		return 0, 0, false
	}
	secs := now.Sub(prevAt).Seconds()
	if secs <= 0 {
		return 0, 0, false
	}
	return float64(curRx-prevRx) / secs, float64(curTx-prevTx) / secs, true
}

func (s *Sampler) read(counter string) (uint64, error) {
	b, err := os.ReadFile(filepath.Join("/sys/class/net", s.iface, "statistics", counter))
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
}
