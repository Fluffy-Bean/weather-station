package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strconv"
)

const version = "0.0.2"

var database *sqlx.DB
var schema = `
CREATE TABLE IF NOT EXISTS weather (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    temperature REAL,
    humidity REAL,
    pressure REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT,
    name TEXT,
    config TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

type Weather struct {
	Id          int     `json:"id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Pressure    float64 `json:"pressure"`
	CreatedAt   string  `json:"created_at" db:"created_at"`
}
type Device struct {
	Id        int    `json:"id"`
	Uuid      string `json:"uuid"`
	Name      string `json:"name"`
	Config    string `json:"config"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

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
	Id       int          `json:"id"`
	Name     string       `json:"name"`
	Config   DeviceConfig `json:"config"`
	Location string       `json:"location"`
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

type ServerResponse struct {
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

func main() {
	var err error
	database, err = sqlx.Open("sqlite3", "./weather.db")
	if err != nil {
		log.Fatal(err)
	}
	database.MustExec(schema)

	// Run HTTP server
	r := gin.Default()
	r.Static("/static", "./public/static")
	r.LoadHTMLGlob("public/*.html")

	r.GET("/", indexPage)

	r.GET("/health", healthGet)

	r.GET("/weather", weatherGet)
	r.POST("/weather", weatherPost)

	r.GET("/devices", devicesGet)
	r.POST("/devices", devicesPost)
	r.PUT("/devices", devicesPut)
	r.DELETE("/devices", devicesDelete)

	log.Fatal(r.Run(":8080"))
}

func indexPage(c *gin.Context) {
	c.HTML(200, "index.html", nil)
}

func healthGet(c *gin.Context) {
	c.JSON(200, ServerResponse{version, "0"})
}

func weatherGet(c *gin.Context) {
	var weather []Weather

	err := database.Select(&weather, "SELECT id, temperature, humidity, pressure, created_at FROM weather ORDER BY created_at DESC;")
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var weatherResponse []WeatherResponse
	for i := range weather {
		weatherResponse = append(weatherResponse, WeatherResponse{weather[i].Id, weather[i].Temperature, weather[i].Humidity, weather[i].Pressure})
	}

	c.JSON(200, weatherResponse)
}

func weatherPost(c *gin.Context) {
	var form WeatherForm
	if err := c.ShouldBind(&form); err != nil {
		fmt.Println("Error binding form")
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	var device Device
	err := database.Get(&device, "SELECT uuid FROM devices WHERE uuid = ? LIMIT 1;", form.Uuid)
	if err != nil {
		fmt.Println("Device does not exist" + form.Uuid)
		c.JSON(403, gin.H{"error": "Device does not exist, check in first"})
		return
	}

	_, err = database.Exec("INSERT INTO weather (temperature, humidity, pressure) VALUES (:temperature, :humidity, :pressure);", form)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, ":3")
}

func devicesGet(c *gin.Context) {
	var devices []Device
	err := database.Select(&devices, "SELECT id, name, config, created_at FROM devices ORDER BY created_at DESC;")
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var deviceResponse []DeviceResponse
	var config DeviceConfig
	for i := range devices {
		if err := json.Unmarshal([]byte(devices[i].Config), &config); err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
		deviceResponse = append(deviceResponse, DeviceResponse{devices[i].Id, devices[i].Name, config, "Living room"})
	}

	c.JSON(200, deviceResponse)
}

func devicesPost(c *gin.Context) {
	var form DevicePost
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	var devices []Device
	err := database.Get(&devices, "SELECT * FROM devices WHERE uuid = ?;", form.Uuid)
	if err != nil {
		config, err := json.Marshal(DeviceConfig{form.Version, form.Address})
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}

		_, err = database.Exec("INSERT INTO devices (uuid, name, config) VALUES (?, ?, ?);", form.Uuid, form.Name, string(config))
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
	}

	c.JSON(200, ":3")
}

func devicesPut(c *gin.Context) {
	var form DevicePut
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	_, err := database.Exec("UPDATE devices SET name = ? WHERE id = ?;", form.Name, form.Id)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, ":3")
}

func devicesDelete(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	_, err = database.Exec("DELETE FROM devices WHERE id = ?;", id)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, ":3")
}
