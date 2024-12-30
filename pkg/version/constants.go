package version

const (
	MAX_MAJOR = 20
	MAX_MINOR = 50
	MAX_PATCH = 50
)

type Strategy string

const (
	StrategyDynamic Strategy = "dynamic"
	StrategyExact   Strategy = "exact"
	StrategyRange   Strategy = "range"
)
