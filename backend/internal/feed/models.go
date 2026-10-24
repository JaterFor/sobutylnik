package feed

type Candidate struct {
	UserID          string
	Name            string
	Age             int
	Gender          string
	Bio             string
	ReadinessStatus string
	DistanceMeters  float64
	Photos          []string
}

const (
	DefaultRadiusKm = 5.0
	MaxRadiusKm     = 50.0
	MinRadiusKm     = 0.5
	FeedPageSize    = 20
)
