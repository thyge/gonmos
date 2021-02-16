package nmos

type QuerySubscription struct {
	MaxUpdateRateMs int           `json:"max_update_rate_ms"`
	ResourcePath    string        `json:"resource_path"`
	Params          []interface{} `json:"params"`
	Persist         bool          `json:"persist"`
	Secure          bool          `json:"secure"`
}
