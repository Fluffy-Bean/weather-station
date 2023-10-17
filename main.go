package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strconv"
)

const version = "0.0.1"

var database *sql.DB

type WeatherResponse struct {
	Id          int     `json:"id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Pressure    float64 `json:"pressure"`
}
type WeatherForm struct {
	Uuid        string  `form:"uuid" json:"uuid" binding:"required"`
	Temperature float64 `form:"temperature" json:"temperature" binding:"required"`
	Humidity    float64 `form:"humidity" json:"humidity" binding:"required"`
	Pressure    float64 `form:"pressure" json:"pressure" binding:"required"`
}

type DeviceResponse struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Config   string `json:"config"`
	Location string `json:"location"`
}
type DeviceConfig struct {
	Version string `json:"version"`
	Address string `json:"address"`
}
type DevicePost struct {
	Uuid    string `form:"uuid" json:"uuid" binding:"required"`
	Name    string `form:"name" json:"name" binding:"required"`
	Version string `form:"version" json:"version" binding:"required"`
	Address string `form:"address" json:"address" binding:"required"`
}
type DevicePut struct {
	Id       int    `form:"id" json:"id" binding:"required"`
	Name     string `form:"name" json:"name" binding:"required"`
	Location string `form:"location" json:"location" binding:"required"`
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
	statement, err = database.Prepare("CREATE TABLE IF NOT EXISTS devices (id INTEGER PRIMARY KEY AUTOINCREMENT, uuid TEXT, name TEXT, config TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	// Run HTTP server
	r := gin.Default()
	r.Static("/static", "./public/static")
	r.LoadHTMLGlob("public/*.html")

	r.GET("/", indexPage)

	r.GET("/weather", weatherGet)
	r.POST("/weather", weatherPost)

	r.GET("/devices", devicesGet)
	r.POST("/devices", devicesPost)
	r.PUT("/devices", devicesPut)
	r.DELETE("/devices", devicesDelete)

	log.Fatal(r.Run(":8080"))
}

func indexPage(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.HTML(200, "index.html", nil)
}

func weatherGet(c *gin.Context) {
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
		if err := row.Scan(&id, &temperature, &humidity, &pressure); err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
		responseData = append(responseData, WeatherResponse{id, temperature, humidity, pressure})
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, responseData)
}

func weatherPost(c *gin.Context) {
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
		c.JSON(403, gin.H{"error": "Device does not exist, check in first"})
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

	statement, err := database.Prepare("SELECT id, name, config FROM devices;")
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
		responseData []DeviceResponse
		id           int
		name, config string
	)

	for row.Next() {
		if err := row.Scan(&id, &name, &config); err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
		responseData = append(responseData, DeviceResponse{id, name, config, "Living room"})
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, responseData)
}

func devicesPost(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	var form DevicePost
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	statement, _ := database.Prepare("SELECT id FROM devices WHERE uuid = ? LIMIT 1;")
	row, err := statement.Query(form.Uuid)
	if err != nil {
		fmt.Println("Error checking if device exists")
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	if row.Next() {
		var id int
		if err := row.Scan(&id); err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
	} else {
		config, err := json.Marshal(DeviceConfig{form.Version, form.Address})
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}

		statement, _ = database.Prepare("INSERT INTO devices (uuid, name, config) VALUES (?, ?, ?);")
		_, err = statement.Exec(form.Uuid, form.Name, string(config))
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
	}

	_ = statement.Close()
	_ = row.Close()

	c.JSON(200, ":3")
}

func devicesPut(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	var form DevicePut
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	statement, _ := database.Prepare("UPDATE devices SET name = ? WHERE id = ?;")
	_, err := statement.Exec(form.Name, form.Id)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	_ = statement.Close()

	c.JSON(200, ":3")
}

func devicesDelete(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	statement, _ := database.Prepare("DELETE FROM devices WHERE id = ?;")
	_, err = statement.Exec(id)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	_ = statement.Close()

	c.JSON(200, ":3")
}
