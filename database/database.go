package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
	"shop.cloudsheeptech.com/configuration"
)

// A small database wrapper allowing to access a MySQL database

// ------------------------------------------------------------
// Configuration File Handling
// ------------------------------------------------------------

var config DBConf

type DBConf struct {
	DBUser string
	DBPass string
	Addr   string
	DBName string
}

func createDefaultConfiguration(confFile string) {
	// This method is only meant to create the file in the right format
	// It is not meant to create a config file holding a working configuration
	conf := DBConf{
		DBUser: "",
		DBPass: "",
		Addr:   "127.0.0.1:3306",
		DBName: "shoppinglist",
	}
	config = conf
	storeConfiguration(confFile)
}

func loadConfig(confFile string) {
	if confFile == "" {
		log.Fatal("Cannot load database configuration")
	}
	content, err := os.ReadFile(confFile)
	if err != nil {
		createDefaultConfiguration(confFile)
		log.Fatalf("Failed to read database configuration file: %s", err)
	}
	var configuration DBConf
	err = json.Unmarshal(content, &configuration)
	if err != nil {
		log.Fatalf("Configuration file '%s' not in correct format: %s", confFile, err)
	}
	config = configuration
	log.Printf("Successfully loaded configuration from '%s'", confFile)
}

func storeConfiguration(confFile string) {
	if confFile == "" {
		log.Fatal("Cannot store configuration file due to empty location")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Failed to convert configuration to file format")
	}
	err = os.WriteFile(confFile, encoded, 0660)
	if err != nil {
		log.Fatalf("Failed to store configuration to file: %s", err)
	}
	log.Printf("Stored configuration to file: %s", confFile)
}

// ------------------------------------------------------------

func CheckDatabaseOnline(cfg configuration.Config) {
	if config == (DBConf{}) {
		log.Print("Configuration not initialized")
		loadConfig(cfg.DatabaseConfig)
	}
	mysqlCfg := mysql.Config{
		User:                 config.DBUser,
		Passwd:               config.DBPass,
		Net:                  "tcp",
		Addr:                 config.Addr,
		DBName:               config.DBName,
		AllowNativePasswords: true,
	}
	db, err := sql.Open("mysql", mysqlCfg.FormatDSN())
	if err != nil {
		log.Fatalf("Cannot connect to database: %s", err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatalf("Database not responding: %s", pingErr)
	}
	log.Print("Connected to database")
}

func GetShopList(id int) error {
	if id < 0 {
		err := errors.New("Cannot open empty database table")
		return err
	}
	return nil
}
