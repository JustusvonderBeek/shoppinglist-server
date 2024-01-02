package configuration

type Config struct {
	ListenAddr string
	ListenPort string

	Logfile        string
	DatabaseConfig string
	TLSCertificate string
	TLSKeyfile     string
	JWTSecretFile  string

	JWTTimeout int // This is only meant for testing; not for production
}
