package routes

import (
	"context"
	"log"
	"net/http"

	"buratud.com/evr-sensor-collector-server/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func GetAllRoute(c *gin.Context) {
	cursor, err := dbClient.Database("evr").Collection("routes").Find(context.Background(), bson.D{})
	if err != nil {
		log.Printf("Error fetching routes: %v", err)
		c.String(http.StatusInternalServerError, "Error fetching routes")
		return
	}
	var routes []types.Route = []types.Route{}
	if err := cursor.All(context.Background(), &routes); err != nil {
		log.Printf("Error decoding routes: %v", err)
		c.String(http.StatusInternalServerError, "Error decoding routes")
		return
	}
	c.JSON(http.StatusOK, routes)
}

func GetRoute(c *gin.Context) {
	var routeRequest types.RouteRequest
	if err := c.ShouldBindUri(&routeRequest); err != nil {
		log.Printf("Error binding query: %v", err)
		c.String(http.StatusBadRequest, "Invalid query parameters")
		return
	}
	var route types.Route
	id, err := bson.ObjectIDFromHex(routeRequest.ID)
	if err != nil {
		log.Printf("Error converting ID: %v", err)
		c.String(http.StatusBadRequest, "Invalid ID format")
		return
	}
	if err := dbClient.Database("evr").Collection("routes").FindOne(context.Background(), bson.M{"_id": id}).Decode(&route); err != nil {
		if err == mongo.ErrNoDocuments {
			c.String(http.StatusNotFound, "Route not found")
			return
		}
		log.Printf("Error fetching route: %v", err)
		c.String(http.StatusInternalServerError, "Error fetching route")
		return
	}
	c.JSON(http.StatusOK, route)
}

func PostRoute(c *gin.Context) {
	var route types.Route
	if err := c.ShouldBindJSON(&route); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.String(http.StatusBadRequest, "Invalid JSON")
		return
	}
	if result, err := dbClient.Database("evr").Collection("routes").InsertOne(context.Background(), route); err != nil {
		log.Printf("Error inserting route: %v", err)
		c.String(http.StatusInternalServerError, "Error inserting route")
		return
	} else {
		route.ID = result.InsertedID.(bson.ObjectID)
	}
	c.JSON(http.StatusCreated, route)
}

func PutRoute(c *gin.Context) {
	var routeRequest types.RouteRequest
	if err := c.ShouldBindUri(&routeRequest); err != nil {
		log.Printf("Error binding query: %v", err)
		c.String(http.StatusBadRequest, "Invalid query parameters")
		return
	}
	id, err := bson.ObjectIDFromHex(routeRequest.ID)
	if err != nil {
		log.Printf("Error converting ID: %v", err)
		c.String(http.StatusBadRequest, "Invalid ID format")
		return
	}
	if err := dbClient.Database("evr").Collection("routes").FindOne(context.Background(), bson.M{"_id": id}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			c.String(http.StatusNotFound, "Route not found")
			return
		}
	}
	var route types.Route
	if err := c.ShouldBindJSON(&route); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.String(http.StatusBadRequest, "Invalid JSON")
		return
	}
	if _, err := dbClient.Database("evr").Collection("routes").UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": route}); err != nil {
		log.Printf("Error updating route: %v", err)
		c.String(http.StatusInternalServerError, "Error updating route")
		return
	}
	c.JSON(http.StatusOK, route)
}

func DeleteRoute(c *gin.Context) {
	var routeRequest types.RouteRequest
	if err := c.ShouldBindUri(&routeRequest); err != nil {
		log.Printf("Error binding query: %v", err)
		c.String(http.StatusBadRequest, "Invalid query parameters")
		return
	}
	id, err := bson.ObjectIDFromHex(routeRequest.ID)
	if err != nil {
		log.Printf("Error converting ID: %v", err)
		c.String(http.StatusBadRequest, "Invalid ID format")
		return
	}
	if _, err := dbClient.Database("evr").Collection("routes").DeleteOne(context.Background(), bson.M{"_id": id}); err != nil {
		log.Printf("Error deleting route: %v", err)
		c.String(http.StatusInternalServerError, "Error deleting route")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Route deleted successfully"})
}
