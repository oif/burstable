package burstable_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/oif/burstable"

	"github.com/stretchr/testify/assert"
)

type fakeController struct {
	usage uint64
	quota uint64
	wait  time.Duration
}

// implements controller interface
func (c *fakeController) GetCurrentPeriodUsage() uint64 {
	return c.usage
}

func (c *fakeController) SetNextPriodQuota(quota uint64) {
	c.quota = quota
}

func (c *fakeController) use(x uint64) {
	c.usage = x
	time.Sleep(c.wait)
}

func TestBurster(t *testing.T) {
	period := time.Millisecond * 10
	quota := uint64(1)
	burst := uint64(2)
	controller := &fakeController{
		wait: period + time.Millisecond,
	}

	burster := burstable.New(period, quota, burst, controller)
	assert.Equal(t, uint64(0), burster.GetCredit())
	assert.Equal(t, uint64(0), controller.quota)

	go burster.Run()
	defer burster.Stop()
	controller.use(0)
	// earn 1 credit
	assert.Equal(t, uint64(1), burster.GetCredit())
	// burst 1 credit
	assert.Equal(t, quota+1, controller.quota)
	controller.use(quota)
	// just hit quota, no credit earned
	assert.Equal(t, uint64(1), burster.GetCredit())
	assert.Equal(t, quota+1, controller.quota)

	controller.use(2)
	// used all credit then back to original quota
	assert.Equal(t, uint64(0), burster.GetCredit())
	assert.Equal(t, quota, controller.quota)

	// idle for (burst+1) periods, earned (burst+1) credit but only can burst to quota + burst
	for i := uint64(0); i <= burst; i++ {
		controller.use(0)
	}
	assert.Equal(t, quota+burst, controller.quota)
	assert.Equal(t, burst+1, burster.GetCredit())
	// up to limit
	controller.use(quota + burst)
	assert.Equal(t, uint64(1), burster.GetCredit())
	assert.Equal(t, quota+1, controller.quota)
}

func TestEffectOfBurster(t *testing.T) {
	for _, testCase := range []struct {
		Name  string
		usage []uint64
		quota uint64
		burst uint64
	}{
		{
			Name:  "fixed1",
			usage: []uint64{1, 2, 3, 0, 3, 2, 1},
			quota: 1,
			burst: 1,
		},
		{
			Name:  "fixed2",
			usage: []uint64{1, 2, 3, 0, 3, 2, 1},
			quota: 1,
			burst: 2,
		},
		{
			Name:  "fixed3",
			usage: []uint64{1, 2, 3, 0, 3, 2, 1},
			quota: 2,
			burst: 2,
		},
		{
			Name:  "fixed4",
			usage: []uint64{1, 2, 3, 0, 3, 2, 1},
			quota: 3,
			burst: 3,
		},
		{
			Name:  "random1",
			usage: randomUsage(10, 5),
			quota: 4,
			burst: 3,
		},
		{
			Name:  "random2",
			usage: randomUsage(10, 10),
			quota: 4,
			burst: 3,
		},
	} {
		withoutBurst := simulate(testCase.usage, testCase.quota, 0)
		withBurst := simulate(testCase.usage, testCase.quota, testCase.burst)
		t.Logf("[%s] %v quota(%d) without %s with burst(%d) %s -> %.2fx faster", testCase.Name, testCase.usage, testCase.quota, withoutBurst, testCase.burst, withBurst, float64(withoutBurst.Milliseconds())/float64(withBurst.Milliseconds()))
	}
}

type simulateController struct {
	usage  []uint64
	quota  uint64
	stopCh chan struct{}
}

func (c *simulateController) GetCurrentPeriodUsage() uint64 {
	now := c.usage[0]
	if now <= c.quota {
		c.usage = c.usage[1:]
	} else {
		// exceed quota, have to wait next time
		c.usage[0] = now - c.quota
		now = c.quota
	}
	if len(c.usage) == 0 {
		close(c.stopCh)
	}
	return now
}

func (c *simulateController) SetNextPriodQuota(quota uint64) {
	c.quota = quota
}

func simulate(usage []uint64, quota uint64, burst uint64) time.Duration {
	startAt := time.Now()
	shadow := make([]uint64, len(usage))
	copy(shadow, usage)
	controller := &simulateController{
		usage:  shadow,
		stopCh: make(chan struct{}),
	}
	burster := burstable.New(time.Millisecond*10, quota, burst, controller)
	go burster.Run()
	<-controller.stopCh
	burster.Stop()
	return time.Since(startAt)
}

func randomUsage(size int, max int32) []uint64 {
	slice := make([]uint64, size)
	for i := 0; i < size; i++ {
		slice[i] = uint64(rand.Int31n(max))
	}
	return slice
}
