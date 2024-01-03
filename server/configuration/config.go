package configuration

type Config struct {
	ListenAddr string
	ListenPort string

	Logfile        string
	DatabaseConfig string
	TLSCertificate string
	TLSKeyfile     string
	JWTSecretFile  string

	JWTTimeout float32 // This is only meant for testing; not for production
}
