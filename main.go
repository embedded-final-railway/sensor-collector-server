package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var dbPool *pgxpool.Pool

var tempDir string

func convertToSensorData(record []string) (SensorData, error) {
	data := SensorData{}
	unixTime, err := strconv.ParseFloat(record[0], 64)
	if err != nil {
		log.Printf("Error parsing timestamp: %v", err)
		return data, err
	}
	unixTimeSec := int64(unixTime)
	unixTimeNano := int64((unixTime - float64(unixTimeSec)) * 1e9)
	timestamp := time.Unix(unixTimeSec, unixTimeNano).UTC()
	data.Timestamp = timestamp
	data.AccelX, _ = strconv.ParseFloat(record[1], 64)
	data.AccelY, _ = strconv.ParseFloat(record[2], 64)
	data.AccelZ, _ = strconv.ParseFloat(record[3], 64)
	if record[4] != "" {
		latitude, _ := strconv.ParseFloat(record[4], 64)
		data.Latitude = &latitude
	}
	if record[5] != "" {
		longitude, _ := strconv.ParseFloat(record[5], 64)
		data.Longitude = &longitude
	}
	return data, nil
}

func msgHandler(filePath string) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer func() {
		file.Close()
		if err := os.Remove(filePath); err != nil {
			log.Printf("Error deleting file: %v", err)
		}
	}()

	reader := csv.NewReader(file)
	sensorData := make([]SensorData, 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading CSV: %v", err)
			return
		}
		if len(record) != 0 {
			data, err := convertToSensorData(record)
			if err != nil {
				log.Printf("Error converting to SensorData: %v", err)
				return
			}
			sensorData = append(sensorData, data)
		}
	}
	copyCount, err := dbPool.CopyFrom(
		context.Background(),
		pgx.Identifier{"sensor_data"},
		[]string{"timestamp", "accel_x", "accel_y", "accel_z", "latitude", "longitude"},
		pgx.CopyFromSlice(len(sensorData), func(i int) ([]any, error) {
			return []any{sensorData[i].Timestamp, sensorData[i].AccelX, sensorData[i].AccelY, sensorData[i].AccelZ, sensorData[i].Latitude, sensorData[i].Longitude}, nil
		}),
	)
	if err != nil {
		log.Printf("Error copying data to database: %v", err)
		return
	}
	log.Printf("Inserted %d records into the database\n", copyCount)
}

func uploadHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		c.String(http.StatusInternalServerError, "Failed to read request body")
		return
	}
	defer c.Request.Body.Close()
	now := time.Now()
	filename := fmt.Sprintf("%s-%03d.txt", now.Format("20060102150405"), rand.Intn(1000)) // yyyymmddhhmmss
	filePath := filepath.Join(tempDir, filename)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		c.String(http.StatusInternalServerError, "Failed to create file")
		return
	}
	defer file.Close()

	n, err := file.Write(body)
	if err != nil {
		log.Printf("Failed to write to file: %v", err)
		c.String(http.StatusInternalServerError, "Failed to write body to file")
		return
	}

	go msgHandler(filePath) // Start processing the file in a goroutine
	c.Status(http.StatusOK)
	log.Printf("Received %d bytes and saved to %s\n", n, filePath) // Print the full path
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dbPool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_CONNECTION_STRING"))
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	tempDir = filepath.Join(os.TempDir(), "evr-sensor-collector-server")
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
}

func getSensorData(c *gin.Context) {
	var query SensorDataRequest = SensorDataRequest{
		Size: 500,
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		log.Printf("Error binding query: %v", err)
		c.String(http.StatusBadRequest, "Invalid query parameters")
		return
	}
	rows, err := dbPool.Query(context.Background(), "SELECT * FROM (SELECT * FROM sensor_data ORDER BY timestamp DESC LIMIT $1) subquery ORDER BY timestamp ASC", query.Size)
	if err != nil {
		log.Printf("Error querying database: %v", err)
		c.String(http.StatusInternalServerError, "Error querying database")
		return
	}
	responseData, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (SensorData, error) {
		data := SensorData{}
		err := row.Scan(&data.ID, &data.Timestamp, &data.AccelX, &data.AccelY, &data.AccelZ, &data.Latitude, &data.Longitude)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			return data, err
		}
		return data, nil
	})
	if err != nil {
		log.Printf("Error collecting rows: %v", err)
		c.String(http.StatusInternalServerError, "Error collecting rows")
		return
	}
	c.JSON(http.StatusOK, responseData)
}

func main() {
	router := gin.Default()

	router.POST("/upload", uploadHandler)
	router.GET("/sensor_data", getSensorData)
	log.Println("Starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
