package burstable

import (
	"sync/atomic"
	"time"
)

var _ Burster = new(burster)

type burster struct {
	credit *uint64
	stopCh chan struct{}
	// settings
	period     time.Duration
	quota      uint64
	burst      uint64
	controller Controller
}

func New(period time.Duration, quota uint64, burst uint64, controller Controller) *burster {
	b := &burster{
		credit:     new(uint64),
		stopCh:     make(chan struct{}),
		period:     period,
		quota:      quota,
		burst:      burst,
		controller: controller,
	}
	return b
}

func (b *burster) Run() {
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
			if used > b.quota {
				// indicates use burst in this period, should take out from credit
				overrun := used - b.quota
				if overrun > b.burst {
					overrun = b.burst
				}
				// if more than credit
				if credit := atomic.LoadUint64(b.credit); overrun > credit {
					atomic.StoreUint64(b.credit, 0)
				} else {
					atomic.AddUint64(b.credit, ^uint64(overrun-1))
				}
			} else {
				atomic.AddUint64(b.credit, b.quota-used)
			}
			nextBurst := atomic.LoadUint64(b.credit)
			if nextBurst > b.burst {
				nextBurst = b.burst
			}
			b.controller.SetNextPriodQuota(b.quota + nextBurst)
		}
	}
}

func (b *burster) GetCredit() uint64 {
	return atomic.LoadUint64(b.credit)
}

func (b *burster) Stop() {
	close(b.stopCh)
}
