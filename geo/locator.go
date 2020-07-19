package geo

import (
	"math"
	"math/rand"
)

const (
	maxCoordinate = 1000
)

type Locator interface {
	Coordinates() (float64, float64)
	Distance(x, y float64) float64
	Update()
}

func RandomLocator() Locator {
	return &locator{
		x: randCoordinate(maxCoordinate),
		y: randCoordinate(maxCoordinate),
	}
}

func randCoordinate(limit int) float64 {
	return float64(rand.Intn(limit*2+1) - limit)
}

type locator struct {
	x float64
	y float64
}

func (l *locator) Coordinates() (float64, float64) {
	return l.x, l.y
}

// Distance returns the distance from station to the given coordinates
func (l *locator) Distance(x, y float64) float64 {
	dx := l.x - x
	dy := l.y - y
	return math.Sqrt(dx*dx + dy*dy)
}

// Set changes locator's coordinates
func (l *locator) Update() {
	l.x = randCoordinate(maxCoordinate)
	l.y = randCoordinate(maxCoordinate)
}
