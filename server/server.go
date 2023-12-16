package server

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"shop.cloudsheeptech.com/configuration"
)

func getShopList(c *gin.Context) {
	sId := c.Param("id")
	id, err := strconv.Atoi(sId)
	if err != nil {
		log.Printf("Failed to parse given list id: %s", sId)
		log.Printf("Err: %s", err)
		return
	}
	log.Printf("Trying to access list: %d", id)

}

func Start(cfg configuration.Config) error {
	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	router.GET("/list/:id", getShopList)

	address := cfg.ListenAddr + ":" + cfg.ListenPort
	router.Run(address)

	return nil
}
