package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strconv"
)

var database *sql.DB

type Response struct {
	Data interface{}
}
type WeatherResponse struct {
	Id          int
	Temperature float64
	Humidity    float64
	Pressure    float64
}
type DeviceResponse struct {
	Id       int
	Name     string
	LastSeen string
	Address  string
	Version  string
	Location string
}
type ErrorResponse struct {
	Message string
	Code    int
}

func main() {
	var err error
	database, err = sql.Open("sqlite3", "./weather.db")
	if err != nil {
		log.Fatal(err)
	}

	CreateTables()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/devices", handleDevices)

	err = http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		log.Fatal(err)
	}
	err = database.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func CreateTables() {
	fmt.Println("Creating tables...")
	var (
		statement *sql.Stmt
		err       error
	)

	statement, err = database.Prepare("CREATE TABLE IF NOT EXISTS weather (id INTEGER PRIMARY KEY AUTOINCREMENT, temperature REAL, humidity REAL, pressure REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	statement, err = database.Prepare("CREATE TABLE IF NOT EXISTS devices (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, version TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}
}

func handleRoot(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("Request for /")
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	var err error

	switch request.Method {
	case "GET":
		handleRootGet(writer, request)
	case "POST":
		handleRootPost(writer, request)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
		err = json.NewEncoder(writer).Encode(Response{ErrorResponse{"Method not allowed", 405}})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleRootGet(writer http.ResponseWriter, request *http.Request) {
	var (
		err          error
		statement    *sql.Stmt
		rows         *sql.Rows
		responseData []WeatherResponse
		id           int
		temperature  float64
		humidity     float64
		pressure     float64
	)

	statement, err = database.Prepare("SELECT id, temperature, humidity, pressure FROM weather ORDER BY created_at DESC;")
	if err != nil {
		handleError(writer, request, err)
	}
	rows, err = statement.Query()
	if err != nil {
		handleError(writer, request, err)
	}

	for rows.Next() {
		err = rows.Scan(&id, &temperature, &humidity, &pressure)
		handleError(writer, request, err)
		responseData = append(responseData, WeatherResponse{id, temperature, humidity, pressure})
	}

	err = json.NewEncoder(writer).Encode(Response{responseData})
	if err != nil {
		log.Fatal(err)
	}
}

func handleRootPost(writer http.ResponseWriter, request *http.Request) {
	var (
		err         error
		temperature float64
		humidity    float64
		pressure    float64
	)

	temperature, err = strconv.ParseFloat(request.URL.Query().Get("temperature"), 64)
	if err != nil {
		handleError(writer, request, err)
	}
	humidity, err = strconv.ParseFloat(request.URL.Query().Get("humidity"), 64)
	if err != nil {
		handleError(writer, request, err)
	}
	pressure, err = strconv.ParseFloat(request.URL.Query().Get("pressure"), 64)
	if err != nil {
		handleError(writer, request, err)
	}
	statement, _ := database.Prepare("INSERT INTO weather (temperature, humidity, pressure) VALUES (?, ?, ?);")
	_, err = statement.Exec(temperature, humidity, pressure)
	if err != nil {
		handleError(writer, request, err)
	}

	err = json.NewEncoder(writer).Encode(Response{WeatherResponse{0, temperature, humidity, pressure}})
	if err != nil {
		log.Fatal(err)
	}
}

func handleDevices(writer http.ResponseWriter, request *http.Request) {
	var err error

	fmt.Println("Request for /devices")
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	switch request.Method {
	case "GET":
		handleDeviceGet(writer, request)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
		err = json.NewEncoder(writer).Encode(Response{ErrorResponse{"Method not allowed", 405}})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleDeviceGet(writer http.ResponseWriter, request *http.Request) {
	var (
		err          error
		responseData []DeviceResponse
		id           int
		name         string
		version      string
	)

	statement, err := database.Prepare("SELECT id, name, version FROM devices;")
	if err != nil {
		handleError(writer, request, err)
	}
	rows, err := statement.Query()
	if err != nil {
		handleError(writer, request, err)
	}

	for rows.Next() {
		err = rows.Scan(&id, &name, &version)
		if err != nil {
			handleError(writer, request, err)
		}
		responseData = append(responseData, DeviceResponse{id, name, "1 hour ago", "192.168.0.69", "1.0.0", "Living room"})
	}

	err = json.NewEncoder(writer).Encode(Response{responseData})
	if err != nil {
		log.Fatal(err)
	}
}

func handleError(writer http.ResponseWriter, request *http.Request, err error) {
	request.Header.Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusInternalServerError)
	err = json.NewEncoder(writer).Encode(Response{ErrorResponse{"Internal server error", 500}})
	if err != nil {
		log.Fatal(err)
	}
}
