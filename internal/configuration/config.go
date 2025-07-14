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
	// Server configuration
	addr := flag.String("a", "0.0.0.0", "Listen address")
	port := flag.String("p", "46152", "Listen port")

	// TLS Configuration
	tlscert := flag.String("cert", "resources/shop.cloudsheeptech.com.crt", "The location of the TLS TLSCertificateFile")
	tlskey := flag.String("key", "resources/shop.cloudsheeptech.com.pem", "The location of the TLS keyfile")
	noTls := flag.Bool("k", false, "Disable TLS for testing")

	// Authentication configuration
	jwtFile := flag.String("jwt", "resources/jwtSecret.json", "The path to the file holding the JWT Secret")

	// Database configuration
	dbConfig := flag.String("c", "resources/db.json", "The database configuration file")
	resetDb := flag.Bool("reset", false, "Reset the whole database")

	// Auxiliary config
	logfile := flag.String("l", "server.log", "The logfile location")
	production := flag.Bool("production", false, "Enable production mode")

	flag.Parse()

	// Take environment options first, overwrite by command-line options
	storedDatabaseConfig, err := LoadDatabaseConfig(*dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	osDbHost, envExists := os.LookupEnv("DB_HOST")
	if !envExists {
		osDbHost = storedDatabaseConfig.DatabaseHost
	}
	osDbName, envExists := os.LookupEnv("DB_NAME")
	if !envExists {
		osDbName = storedDatabaseConfig.DatabaseName
	}
	osDbUser, envExists := os.LookupEnv("DB_USER")
	if !envExists {
		osDbUser = storedDatabaseConfig.DatabaseUser
	}
	osDbPassword, envExists := os.LookupEnv("DB_PASSWORD")
	if !envExists {
		osDbPassword = storedDatabaseConfig.DatabasePassword
	}

	// Production environment variable
	osProductionBool, envExists := os.LookupEnv("PRODUCTION")
	osProduction := *production
	if envExists {
		osProduction, _ = strconv.ParseBool(osProductionBool)
	}

	serverConfig := ServerConfig{
		ListenAddr: *addr,
		ListenPort: *port,
	}
	tlsConfig := TLSConfig{
		TLSCertificateFile: *tlscert,
		TLSKeyFile:         *tlskey,
		DisableTLS:         *noTls,
	}
	databaseConfig := DatabaseConfig{
		DatabaseConfigFile:  *dbConfig,
		DatabaseUser:        osDbUser,
		DatabasePassword:    osDbPassword,
		DatabaseName:        osDbName,
		DatabaseHost:        osDbHost,
		DatabaseNetworkType: "tcp",
		ResetDatabase:       *resetDb,
	}
	authConfig := AuthConfig{
		JwtSecretFile: *jwtFile,
		JwtTimeout:    500,
	}

	config := Config{
		ServerAddrConfig: serverConfig,
		TLSConfig:        tlsConfig,
		DatabaseConfig:   databaseConfig,
		JwtConfig:        authConfig,
		Production:       osProduction,
		Logfile:          *logfile,
	}

	return config
}

func LoadDatabaseConfig(filename string) (DatabaseConfig, error) {
	if filename == "" {
		return DatabaseConfig{}, errors.New("no database config file given")
	}
	content, err := loadConfigFile(filename)
	if err != nil && os.IsNotExist(err) {
		createDefaultConfiguration(filename)
		return DatabaseConfig{}, errors.New("no config file found, created default one but missing entries")
	} else if err != nil {
		return DatabaseConfig{}, err
	}
	var conf DatabaseConfig
	err = json.Unmarshal(content, &conf)
	if err != nil {
		return DatabaseConfig{}, err
	}
	return conf, nil
}

func loadConfigFile(filename string) ([]byte, error) {
	return util.ReadFileFromRoot(filename)
}

// This method is only meant to create the file in the right format
// It is not meant to create a config file holding a working configuration
func createDefaultConfiguration(confFile string) {
	conf := DatabaseConfig{
		DatabaseConfigFile:  confFile,
		DatabaseUser:        "username",
		DatabasePassword:    "password",
		DatabaseName:        "shopping-list-prod",
		DatabaseHost:        "localhost",
		DatabaseNetworkType: "tcp",
		ResetDatabase:       false,
	}
	storeConfiguration(confFile, conf)
}

func storeConfiguration(filename string, config DatabaseConfig) {
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
