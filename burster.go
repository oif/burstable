package burstable

import (
	"math"
	"sync"
	"time"
)

var _ Burster = new(burster)

type burster struct {
	lock       sync.RWMutex
	credit     uint64
	stopCh     chan struct{}
	creditCeil uint64
	started    bool
	// settings
	period     time.Duration
	quota      uint64
	burst      uint64
	controller Controller
}

func New(period time.Duration, quota uint64, burst uint64, controller Controller) *burster {
	b := &burster{
		credit:     0,
		stopCh:     make(chan struct{}),
		creditCeil: math.MaxUint64 - quota,
		period:     period,
		quota:      quota,
		burst:      burst,
		controller: controller,
	}
	return b
}

func (b *burster) Run() {
	if b.started {
		panic("Burster already started")
	}
	b.started = true
	// Initialize
	b.controller.SetNextPriodQuota(b.quota)
	timer := time.NewTicker(b.period)
	for {
		select {
		case <-b.stopCh:
			return
		case <-timer.C:
			// Get previoud period usage stat
			used := b.controller.GetCurrentPeriodUsage()
			b.lock.Lock()
			credit := b.credit
			if used > b.quota {
				// indicates use burst in this period, should take out from credit
				overrun := used - b.quota
				if overrun > b.burst {
					overrun = b.burst
				}
				// if more than credit
				if overrun > credit {
					credit = 0
				} else {
					credit -= overrun
				}
			} else {
				// earn credit
				credit += b.quota - used
				if credit > b.creditCeil {
					credit = b.creditCeil
				}
			}
			b.credit = credit
			nextBurst := credit
			b.lock.Unlock()

			if nextBurst > b.burst {
				nextBurst = b.burst
			}
			b.controller.SetNextPriodQuota(b.quota + nextBurst)
		}
	}
}

func (b *burster) GetCredit() uint64 {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.credit
}

func (b *burster) Stop() {
	close(b.stopCh)
}
