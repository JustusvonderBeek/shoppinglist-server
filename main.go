package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"shop.cloudsheeptech.com/database"
	"shop.cloudsheeptech.com/server"
	"shop.cloudsheeptech.com/server/configuration"
)

func main() {

	addr := flag.String("a", "0.0.0.0", "Listen address")
	port := flag.String("p", "46152", "Listen port")
	logfile := flag.String("l", "server.log", "The logfile location")
	dbConfig := flag.String("c", "resources/db.json", "The database configuration file")
	tlscert := flag.String("cert", "resources/shoppinglist.crt", "The location of the TLS Certificate")
	tlskey := flag.String("key", "resources/shoppinglist.pem", "THe location of the TLS keyfile")
	jwtFile := flag.String("jwt", "resources/jwtSecret.json", "The path to the file holding the JWT Secret")
	resetDb := flag.Bool("reset", false, "Reset the whole database")
	flag.Parse()

	configuration := configuration.Config{
		ListenAddr:     *addr,
		ListenPort:     *port,
		DatabaseConfig: *dbConfig,
		ResetDatabase:  *resetDb,
		TLSCertificate: *tlscert,
		TLSKeyfile:     *tlskey,
		JWTSecretFile:  *jwtFile,
		JWTTimeout:     180, // Maybe make this a parameter later
	}

	setupLogger(*logfile)
	// Fails if database not connected
	database.CheckDatabaseOnline(configuration)
	if configuration.ResetDatabase {
		resetDatabase()
	}
	server.Start(configuration)
}

func resetDatabase() {
	log.Printf("ACTIVATED RESET OF DATABASE! THIS CANNOT BE REVERTED!")
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Continue [y/N]: ")
	confirmed, _ := reader.ReadString('\n')
	confirmed = strings.ToLower(confirmed)
	confirmed = strings.TrimSpace(confirmed)
	if confirmed == "y" || confirmed == "yes" {
		log.Print("Proceed to reset database...")
		database.ResetDatabase()
		return
	}
	log.Print("Reset of database aborted")
}

func setupLogger(logfile string) {
	logFile, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0640)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}
