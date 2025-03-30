package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"buratud.com/evr-sensor-collector-server/routes"
)

func main() {
	router := gin.Default()
	router.Use(cors.Default())
	router.POST("/upload", routes.UploadHandler)
	router.GET("/sensor_data", routes.GetSensorData)
	route := router.Group("/route")
	route.GET("/", routes.GetAllRoute)
	route.GET("/:id", routes.GetRoute)
	route.POST("/", routes.PostRoute)
	route.PUT("/:id", routes.PutRoute)
	route.DELETE("/:id", routes.DeleteRoute)
	log.Println("Starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
