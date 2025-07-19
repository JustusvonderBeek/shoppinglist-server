package configuration

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/util"
	"log"
	"os"
	"strconv"
)

func HandleCommandlineAndExportConfiguration() Config {
	// Configuration file location
	cmdConfigFile := flag.String("c", "resources/config.json", "The configuration file")

	// Server configuration
	addr := flag.String("a", "0.0.0.0", "Listen address")
	port := flag.String("p", "46152", "Listen port")
	production := flag.Bool("production", false, "Enable production mode")
	logfile := flag.String("l", "server.log", "The logfile location")

	// Database configuration
	resetDb := flag.Bool("reset", false, "Reset the whole database")

	// TLS Configuration
	tlsCert := flag.String("cert", "resources/shop.cloudsheeptech.com.crt", "The location of the TLS CertificateFile")
	tlsKey := flag.String("key", "resources/shop.cloudsheeptech.com.pem", "The location of the TLS keyfile")
	tlsDisable := flag.Bool("k", false, "Disable TLS for testing")

	// JWT
	jwtTimeoutMs := flag.Int("t", 500, "JWT timeout in milliseconds")

	flag.Parse()

	//  Command-Line over ENVIRONMENT option
	// First, read the envConfigFile file
	configFile := *cmdConfigFile
	envConfigFile, envExists := os.LookupEnv("CONFIG_FILE")
	if envExists {
		configFile = envConfigFile
	}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "c" {
			configFile = *cmdConfigFile
		}
	})
	config, err := loadConfigFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	// Parse ENV
	envDbHost, envExists := os.LookupEnv("DB_HOST")
	if envExists {
		config.Database.Host = envDbHost
	}

	envProduction, envExists := os.LookupEnv("PRODUCTION")
	if envExists {
		envProductionParsed, err := strconv.ParseBool(envProduction)
		if err != nil {
			log.Printf("Error parsing PRODUCTION '%s' as bool: %v", envProduction, err)
		} else {
			config.Server.Production = envProductionParsed
		}
	}

	// Set the config options from cmd if exists
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "addr":
			config.Server.ListenAddr = *addr
		case "port":
			config.Server.ListenPort = *port
		case "production":
			config.Server.Production = *production
		case "l":
			config.Server.Logfile = *logfile
		case "reset":
			config.Database.Reset = *resetDb
		case "cert":
			config.TLS.CertificateFile = *tlsCert
		case "key":
			config.TLS.KeyFile = *tlsKey
		case "k":
			config.TLS.DisableTLS = *tlsDisable
		case "t":
			config.JWT.JwtTimeoutMs = *jwtTimeoutMs
		}
	})

	return config
}

func loadConfigFile(filename string) (Config, error) {
	if filename == "" {
		return Config{}, errors.New("no config file given")
	}
	content, err := os.ReadFile(filename)
	if err != nil && os.IsNotExist(err) {
		createDefaultConfiguration(filename)
		return Config{}, errors.New("no config file found, created default one but missing entries")
	} else if err != nil {
		return Config{}, err
	}
	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

// This method is only meant to create the file in the right format
// It is not meant to create a config file holding a working configuration
func createDefaultConfiguration(configFile string) {
	conf := Config{
		Server:   ServerConfig{},
		Database: DatabaseConfig{},
		TLS:      TLSConfig{},
		JWT:      AuthConfig{},
		API:      APIKeyConfig{},
		Admin:    AdminConfig{},
	}
	storeConfiguration(configFile, conf)
}

func storeConfiguration(filename string, config Config) {
	if filename == "" {
		log.Fatal("Cannot store configuration file due to empty location")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Failed to convert configuration to file format")
	}
	storedFilename, _, err := util.WriteFileAtRoot(encoded, filename, false)
	log.Printf("Stored configuration to file: %s", storedFilename)
}
