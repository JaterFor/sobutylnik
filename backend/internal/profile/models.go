package profile

import "time"

type Profile struct {
	UserID            string
	Name              string
	Age               int
	Gender            string
	Bio               string
	PreferredDrink    *string
	ReadinessStatus   string
	HasLocation       bool
	Latitude          *float64
	Longitude         *float64
	LocationUpdatedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Photo struct {
	ID        string
	UserID    string
	ObjectKey string
	Position  int
	IsPrimary bool
	CreatedAt time.Time
}

const (
	MaxPhotos    = 3
	MaxBioLength = 500
	MinAge       = 18
)

var validGenders = map[string]bool{"male": true, "female": true, "other": true}
var validReadinessStatuses = map[string]bool{"now": true, "today": true, "this_week": true}

func IsValidGender(g string) bool {
	return validGenders[g]
}

func IsValidReadinessStatus(s string) bool {
	return validReadinessStatuses[s]
}
