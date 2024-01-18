package healthcheck

type HealthcheckType struct {
	App           string `json:"app"`
	Uptime        string `json:"uptime"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Version       string `json:"version"`
}
