package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/JustusvonderBeek/shoppinglist-server/internal/configuration"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/database"
	"github.com/JustusvonderBeek/shoppinglist-server/internal/server"
)

func main() {
	config := configuration.HandleCommandlineAndExportConfiguration()

	setupLogger(config.Server.Logfile)
	// Fails if database not connected
	database.CheckDatabaseOnline(config.Database)
	if config.Database.Reset {
		resetDatabase()
	}
	server.Start(config)
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
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
