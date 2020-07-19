package geo

const (
	_ = iota
	MethodChangePosition
	MethodGetDistance
)

type Request struct {
	Method int
	X      float64
	Y      float64
}

type ServiceDist struct {
	Service string
	Dist    float64
}
