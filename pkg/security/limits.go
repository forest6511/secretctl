package security

// Edition represents the product edition (Free or Team).
type Edition int

const (
	// EditionFree is the free tier with limited security features.
	EditionFree Edition = iota
	// EditionTeam is the team tier with full security features.
	EditionTeam
)

// String returns the edition name.
func (e Edition) String() string {
	switch e {
	case EditionFree:
		return "Free"
	case EditionTeam:
		return "Team"
	default:
		return "Unknown"
	}
}

// Limits defines the feature limits for each edition.
type Limits struct {
	// DuplicateLimit is the max duplicates to show (0 = unlimited).
	DuplicateLimit int
	// WeakLimit is the max weak passwords to show (0 = unlimited).
	WeakLimit int
	// ExportEnabled indicates if security reports can be exported.
	ExportEnabled bool
}

// GetLimits returns the feature limits for the given edition.
func GetLimits(edition Edition) Limits {
	switch edition {
	case EditionFree:
		return Limits{
			DuplicateLimit: 3,
			WeakLimit:      3,
			ExportEnabled:  false,
		}
	case EditionTeam:
		return Limits{
			DuplicateLimit: 0, // Unlimited
			WeakLimit:      0,
			ExportEnabled:  true,
		}
	default:
		return GetLimits(EditionFree)
	}
}

// IsLimited returns true if the results should be limited (Free edition).
func (l Limits) IsLimited() bool {
	return l.DuplicateLimit > 0 || l.WeakLimit > 0
}
