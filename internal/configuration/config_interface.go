package configuration

type Config struct {
	ServerAddrConfig ServerConfig

	TLSConfig TLSConfig

	DatabaseConfig DatabaseConfig

	JwtConfig AuthConfig

	Production bool
	Logfile    string
}

type ServerConfig struct {
	ListenAddr string
	ListenPort string
}

type TLSConfig struct {
	TLSCertificateFile string
	TLSKeyFile         string

	DisableTLS bool
}

type DatabaseConfig struct {
	DatabaseConfigFile string

	DatabaseUser     string
	DatabasePassword string

	DatabaseName string

	DatabaseHost        string
	DatabaseNetworkType string

	ResetDatabase bool
}

type AuthConfig struct {
	JwtSecretFile string
	JwtTimeout    float32 // This is only meant for testing; not for production
}
