package burstable

type Burster interface {
	Run()
	Stop()
}

type Controller interface {
	GetCurrentPeriodUsage() uint64
	SetNextPriodQuota(quota uint64)
}
