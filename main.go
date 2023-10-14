package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	uuid4 "github.com/google/uuid" // Stupid way to avoid conflict
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var database *sql.DB

type WeatherResponse struct {
	Id          int
	Temperature float64
	Humidity    float64
	Pressure    float64
}
type WeatherForm struct {
	Uuid        string  `form:"uuid" json:"uuid" binding:"required"`
	Temperature float64 `form:"temperature" json:"temperature" binding:"required"`
	Humidity    float64 `form:"humidity" json:"humidity" binding:"required"`
	Pressure    float64 `form:"pressure" json:"pressure" binding:"required"`
}
type DeviceResponse struct {
	Id       int
	Name     string
	Address  string
	Version  string
	Location string
}
type DeviceForm struct {
	Name    string `form:"name" json:"name" binding:"required"`
	Version string `form:"version" json:"version" binding:"required"`
	Address string `form:"address" json:"address" binding:"required"`
}

func main() {
	var err error
	database, err = sql.Open("sqlite3", "./weather.db")
	if err != nil {
		log.Fatal(err)
	}

	// Make Weather table
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS weather (id INTEGER PRIMARY KEY AUTOINCREMENT, temperature REAL, humidity REAL, pressure REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}
	_, _ = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	// Make Devices table
	statement, err = database.Prepare("CREATE TABLE IF NOT EXISTS devices (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, address TEXT, version TEXT, uuid TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	// Run HTTP server
	r := gin.Default()
	r.GET("/", indexGet)
	r.POST("/", indexPost)
	r.GET("/devices", devicesGet)
	r.POST("/devices", devicesPost)

	log.Fatal(r.Run(":8080"))
}

func indexGet(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	statement, err := database.Prepare("SELECT id, temperature, humidity, pressure FROM weather ORDER BY created_at DESC;")
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}
	row, err := statement.Query()
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
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

	var form WeatherForm
	if err := c.ShouldBind(&form); err != nil {
		fmt.Println("Error binding form")
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	statement, err := database.Prepare("SELECT id FROM devices WHERE uuid = ? LIMIT 1;")
	row, err := statement.Query(form.Uuid)
	if err != nil {
		fmt.Println("Error checking if device exists")
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}
	if !row.Next() {
		fmt.Println("Device does not exist" + form.Uuid)
		c.JSON(403, gin.H{"error": "Device does not exist"})
		return
	}

	_ = statement.Close()
	_ = row.Close()

	statement, _ = database.Prepare("INSERT INTO weather (temperature, humidity, pressure) VALUES (?, ?, ?);")
	_, err = statement.Exec(form.Temperature, form.Humidity, form.Pressure)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, WeatherResponse{0, form.Temperature, form.Humidity, form.Pressure})
}

func devicesGet(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	statement, err := database.Prepare("SELECT id, name, address, version FROM devices;")
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}
	row, err := statement.Query()
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
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

	var form DeviceForm
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	statement, err := database.Prepare("SELECT uuid FROM devices WHERE address = ? LIMIT 1;")
	row, err := statement.Query(form.Address)
	if err != nil {
		fmt.Println("Error checking if device exists")
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var uuid string
	if row.Next() {
		err = row.Scan(&uuid)
		if err != nil {
			fmt.Println("Error scanning row")
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
	} else {
		uuid = uuid4.NewString()
		statement, _ = database.Prepare("INSERT INTO devices (name, version, address, uuid) VALUES (?, ?, ?, ?);")
		_, err = statement.Exec(form.Name, form.Version, form.Address, uuid)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, gin.H{"uuid": uuid})
}
