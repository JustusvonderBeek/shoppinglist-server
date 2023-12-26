package main

import (
	"flag"
	"io"
	"log"
	"os"

	"shop.cloudsheeptech.com/configuration"
	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server"
)

func main() {

	addr := flag.String("a", "0.0.0.0", "Listen address")
	port := flag.String("p", "46152", "Listen port")
	logfile := flag.String("l", "server.log", "The logfile location")
	dbConfig := flag.String("c", "db.json", "The database configuration file")
	tlscert := flag.String("cert", "resources/shoppinglist.crt", "The location of the TLS Certificate")
	tlskey := flag.String("key", "resources/shoppinglist.pem", "THe location of the TLS keyfile")
	jwtFile := flag.String("jwt", "resources/jwt.secret", "The path to the file holding the JWT Secret")
	flag.Parse()

	configuration := configuration.Config{
		ListenAddr:     *addr,
		ListenPort:     *port,
		DatabaseConfig: *dbConfig,
		TLSCertificate: *tlscert,
		TLSKeyfile:     *tlskey,
		JWTSecretFile:  *jwtFile,
	}

	setupLogger(*logfile)
	// Fails if database not connected
	database.CheckDatabaseOnline(configuration)
	server.Start(configuration)
}

func setupLogger(logfile string) {
	logFile, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0640)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}
