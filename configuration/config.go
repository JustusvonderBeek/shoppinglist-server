package configuration

type Config struct {
	ListenAddr string
	ListenPort string

	Logfile        string
	DatabaseConfig string
	TLSCertificate string
	TLSKeyfile     string
}
