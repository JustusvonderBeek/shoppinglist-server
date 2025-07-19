package configuration

import "time"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	TLS      TLSConfig
	JWT      AuthConfig
	API      APIKeyConfig
	Admin    AdminConfig
}

type ServerConfig struct {
	ListenAddr string
	ListenPort string
	Production bool
	Logfile    string
}

type TLSConfig struct {
	CertificateFile string
	KeyFile         string

	DisableTLS bool
}

type DatabaseConfig struct {
	User     string
	Password string

	Name        string
	Host        string
	NetworkType string

	Reset bool
}

type AuthConfig struct {
	Secret        string
	JwtSecretFile string
	JwtTimeoutMs  int // This is only meant for testing; not for production
}

type APIKeyConfig struct {
	Key        string
	ValidUntil time.Time
}

type AdminConfig struct {
	User     string
	Password string
}
