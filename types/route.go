package types

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Route struct {
	ID          bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string        `json:"name"`
	RoutePoints []Location    `json:"route_points"`
}

type RouteRequest struct {
	ID string `json:"id" form:"id" uri:"id"`
}
