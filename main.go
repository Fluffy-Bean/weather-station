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

const version = "0.0.3"

var database *sqlx.DB
var schema = `
CREATE TABLE IF NOT EXISTS weather (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    temperature REAL NOT NULL,
    humidity REAL NOT NULL,
    pressure REAL NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    uuid TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    room_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (room_id) REFERENCES rooms(id)
);
CREATE TABLE IF NOT EXISTS rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- if nothing in rooms, add default location
INSERT INTO rooms (name) SELECT 'Living room' WHERE NOT EXISTS (SELECT 1 FROM rooms);
`

type Weather struct {
	Id          int     `json:"id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Pressure    float64 `json:"pressure"`
	CreatedAt   string  `json:"created_at" db:"created_at"`
}
type Device struct {
	Id        int     `json:"id"`
	Uuid      string  `json:"uuid"`
	Name      string  `json:"name"`
	Config    string  `json:"config"`
	RoomId    *string `json:"room_id" db:"room_id"`
	CreatedAt string  `json:"created_at" db:"created_at"`
}
type Room struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

type ServerResponse struct {
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
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
	Id   int    `form:"id" json:"id" binding:"required"`
	Name string `form:"name" json:"name" binding:"required"`
	Room string `form:"room" json:"room" binding:"required"`
}

type RoomResponse struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	DeviceCount int    `json:"device_count" db:"device_count"`
}
type RoomPost struct {
	Id   int    `form:"id" json:"id" binding:"required"`
	Name string `form:"name" json:"name" binding:"required"`
}

func main() {
	var err error

	database, err = sqlx.Open("sqlite3", "./weather.db")
	if err != nil {
		log.Fatal(err)
	}
	database.MustExec(schema)

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

	r.GET("/rooms", roomsGet)
	r.POST("/rooms", roomsPost)
	r.PUT("/rooms", roomsPut)
	r.DELETE("/rooms", roomsDelete)

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
	var rooms []Room
	err := database.Select(&rooms, "SELECT * FROM rooms;")
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var devices []Device
	err = database.Select(&devices, "SELECT * FROM devices ORDER BY room_id, created_at DESC;")
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var deviceResponse []gin.H
	var config DeviceConfig
	for i := range devices {
		if err := json.Unmarshal([]byte(devices[i].Config), &config); err != nil {
			fmt.Println(err)
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
		response := gin.H{
			"id":      devices[i].Id,
			"name":    devices[i].Name,
			"room_id": devices[i].RoomId,
			"config":  config,
		}
		deviceResponse = append(deviceResponse, response)
	}

	c.JSON(200, gin.H{"devices": deviceResponse, "rooms": rooms})
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

	_, err := database.Exec("UPDATE devices SET name = ?, room_id = ? WHERE id = ?;", form.Name, form.Room, form.Id)
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

func roomsGet(c *gin.Context) {
	var roomResponse []RoomResponse
	err := database.Select(&roomResponse, "SELECT r.id, r.name, count(d.id) as device_count from rooms as r left join devices as d on d.room_id = r.id group by r.id")
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, roomResponse)
}

func roomsPost(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	_, err := database.Exec("INSERT INTO rooms (name) VALUES (?);", name)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, ":3")
}

func roomsPut(c *gin.Context) {
	var form RoomPost
	if err := c.ShouldBind(&form); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	_, err := database.Exec("UPDATE rooms SET name = ? WHERE id = ?;", form.Name, form.Id)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, ":3")
}

func roomsDelete(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	_, err = database.Exec("DELETE FROM rooms WHERE id = ?;", id)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	_, err = database.Exec("UPDATE devices SET room_id = NULL WHERE room_id = ?;", id)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, ":3")
}
