package skillhubd

// Config contains daemon process settings supplied by its executable entry.
type Config struct {
	Host      string
	Port      int
	SecretKey string
}
