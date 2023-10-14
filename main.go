package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	uuid4 "github.com/google/uuid" // Stupid way to avoid conflict
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strconv"
)

var database *sql.DB

type WeatherResponse struct {
	Id          int
	Temperature float64
	Humidity    float64
	Pressure    float64
}
type DeviceResponse struct {
	Id       int
	Name     string
	Address  string
	Version  string
	Location string
}

func main() {
	var err error
	database, err = sql.Open("sqlite3", "./weather.db")
	if err != nil {
		log.Fatal(err)
	}
	CreateTables()

	r := gin.Default()
	r.GET("/", indexGet)
	r.POST("/", indexPost)
	r.GET("/devices", devicesGet)
	r.POST("/devices", devicesPost)

	log.Fatal(r.Run())
}

func CreateTables() {
	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS weather (id INTEGER PRIMARY KEY AUTOINCREMENT, temperature REAL, humidity REAL, pressure REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	statement, err = database.Prepare("CREATE TABLE IF NOT EXISTS devices (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, address TEXT, version TEXT, uuid TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}
}

func indexGet(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	statement, err := database.Prepare("SELECT id, temperature, humidity, pressure FROM weather ORDER BY created_at DESC;")
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
	}
	row, err := statement.Query()
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
	}

	var (
		responseData                    []WeatherResponse
		id                              int
		temperature, humidity, pressure float64
	)

	for row.Next() {
		err = row.Scan(&id, &temperature, &humidity, &pressure)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			break
		}
		responseData = append(responseData, WeatherResponse{id, temperature, humidity, pressure})
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, responseData)
}

func indexPost(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	uuid := c.PostForm("uuid")

	statement, err := database.Prepare("SELECT id FROM devices WHERE uuid = ? LIMIT 1;")
	row, err := statement.Query(uuid)
	if err != nil {
		fmt.Println("Error checking if device exists")
		c.JSON(500, gin.H{"error": "Internal server error"})
	}
	if !row.Next() {
		fmt.Println("Device does not exist")
		c.JSON(403, gin.H{"error": "Device does not exist"})
	}

	_ = statement.Close()
	_ = row.Close()

	temperature, err := strconv.ParseFloat(c.PostForm("temperature"), 64)
	if err != nil {
		fmt.Printf("Error parsing temperature %v\n", temperature)
		c.JSON(400, gin.H{"error": "Bad Request"})
	}

	humidity, err := strconv.ParseFloat(c.PostForm("humidity"), 64)
	if err != nil {
		fmt.Println("Error parsing humidity " + c.PostForm("humidity"))
		c.JSON(400, gin.H{"error": "Bad Request"})
	}

	pressure, err := strconv.ParseFloat(c.PostForm("pressure"), 64)
	if err != nil {
		fmt.Println("Error parsing pressure " + c.PostForm("pressure"))
		c.JSON(400, gin.H{"error": "Bad Request"})
	}

	statement, _ = database.Prepare("INSERT INTO weather (temperature, humidity, pressure) VALUES (?, ?, ?);")
	_, err = statement.Exec(temperature, humidity, pressure)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, WeatherResponse{0, temperature, humidity, pressure})
}

func devicesGet(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	statement, err := database.Prepare("SELECT id, name, address, version FROM devices;")
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
	}
	row, err := statement.Query()
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
	}

	var (
		responseData           []DeviceResponse
		id                     int
		name, address, version string
	)

	for row.Next() {
		err = row.Scan(&id, &name, &address, &version)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			break
		}
		responseData = append(responseData, DeviceResponse{id, name, address, version, "Living room"})
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, responseData)
}

func devicesPost(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	name := c.PostForm("name")
	version := c.PostForm("version")
	address := c.PostForm("address")

	statement, err := database.Prepare("SELECT uuid FROM devices WHERE address = ? LIMIT 1;")
	row, err := statement.Query(address)
	if err != nil {
		fmt.Println("Error checking if device exists")
		c.JSON(500, gin.H{"error": "Internal server error"})
	}

	var uuid string

	if row.Next() {
		err = row.Scan(&uuid)
		if err != nil {
			fmt.Println("Error scanning row")
			c.JSON(500, gin.H{"error": "Internal server error"})
		}
	} else {
		uuid = uuid4.NewString()
		statement, _ = database.Prepare("INSERT INTO devices (name, version, address, uuid) VALUES (?, ?, ?, ?);")
		_, err = statement.Exec(name, version, address, uuid)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
		}
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, gin.H{"uuid": uuid})
}
