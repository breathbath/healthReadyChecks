package health

// Checker abstracts health checking logic
type Checker interface {
	IsHealthy() (bool, string)
	SubscribeToUnhealthyChange(sf func(reason string))
}
