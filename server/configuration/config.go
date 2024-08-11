package configuration

type Config struct {
	ListenAddr string
	ListenPort string

	Logfile        string
	DatabaseConfig string
	ResetDatabase  bool
	TLSCertificate string
	TLSKeyfile     string
	JWTSecretFile  string
	DisableTLS     bool

	JWTTimeout float32 // This is only meant for testing; not for production
}
