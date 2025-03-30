package routes

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
	"strings"
	"time"

	"buratud.com/evr-sensor-collector-server/types"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)


var dbPool *pgxpool.Pool
var dbClient *mongo.Client

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dbPool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_CONNECTION_STRING"))
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	dbClient, _ = mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_CONNECTION_STRING")))
}

var tempDir string

func init() {
	tempDir = filepath.Join(os.TempDir(), "evr-sensor-collector-server")
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
}

func GetSensorData(c *gin.Context) {
	var query types.SensorDataRequest = types.SensorDataRequest{
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
	responseData, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (types.SensorData, error) {
		data := types.SensorData{}
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

func convertToSensorData(record []string) (types.SensorData, error) {
	data := types.SensorData{}

	unixTimePart := strings.Split(record[0], ".")
	if len(unixTimePart) != 2 {
		log.Printf("Invalid timestamp format: %s", record[0])
		return data, fmt.Errorf("invalid timestamp format")
	}
	if len(unixTimePart[1]) < 6 {
		unixTimePart[1] = strings.Repeat("0", 6-len(unixTimePart[1])) + unixTimePart[1]
	}
	if len(unixTimePart[1]) < 9 {
		unixTimePart[1] += strings.Repeat("0", 9-len(unixTimePart[1]))
	}
	unixTimeSec, err := strconv.ParseInt(unixTimePart[0], 10, 64)
	if err != nil {
		log.Printf("Error parsing timestamp seconds: %v", err)
		return data, err
	}
	unixTimeNano, err := strconv.ParseInt(unixTimePart[1], 10, 64)
	if err != nil {
		log.Printf("Error parsing timestamp nanoseconds: %v", err)
		return data, err
	}
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
	sensorData := make([]types.SensorData, 0)
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
	_, err = dbPool.CopyFrom(
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
	// log.Printf("Inserted %d records into the database\n", copyCount)
}

func UploadHandler(c *gin.Context) {
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

	_, err = file.Write(body)
	if err != nil {
		log.Printf("Failed to write to file: %v", err)
		c.String(http.StatusInternalServerError, "Failed to write body to file")
		return
	}

	go msgHandler(filePath) // Start processing the file in a goroutine
	c.Status(http.StatusOK)
	// log.Printf("Received %d bytes and saved to %s\n", n, filePath) // Print the full path
}

