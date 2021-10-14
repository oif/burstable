package burstable

type Burster interface {
	Run()
}

type Controller interface {
	GetCurrentPeriodUsage() uint64
	SetNextPriodQuota(quota uint64)
}
