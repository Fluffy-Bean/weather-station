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

var Database *sql.DB

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
	Id    int
	Name  string
	Model string
}
type ErrorResponse struct {
	Message string
	Code    int
}

func main() {
	database, _ := sql.Open("sqlite3", "./weather.db")
	Database = database

	CreateTables()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/devices", handleDevices)

	log.Fatal(http.ListenAndServe("localhost:8080", mux))
}

func CreateTables() {
	fmt.Println("Creating tables...")
	weatherTable, _ := Database.Prepare("CREATE TABLE IF NOT EXISTS weather (id INTEGER PRIMARY KEY, temperature REAL, humidity REAL, pressure REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)")
	devicesTable, _ := Database.Prepare("CREATE TABLE IF NOT EXISTS devices (id INTEGER PRIMARY KEY, name TEXT, model TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)")
	weatherTable.Exec()
	devicesTable.Exec()
}

func handleRoot(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("Request for /")
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	switch request.Method {
	case "GET":
		handleRootGet(writer, request)
	case "POST":
		handleRootPost(writer, request)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(writer).Encode(Response{ErrorResponse{"Method not allowed", 405}})
	}
}

func handleRootGet(writer http.ResponseWriter, request *http.Request) {
	statement, _ := Database.Prepare("SELECT id, temperature, humidity, pressure FROM weather ORDER BY created_at DESC")
	rows, _ := statement.Query()

	var (
		responseData []WeatherResponse
		id           int
		temperature  float64
		humidity     float64
		pressure     float64
	)

	for rows.Next() {
		rows.Scan(&id, &temperature, &humidity, &pressure)
		responseData = append(responseData, WeatherResponse{id, temperature, humidity, pressure})
	}

	json.NewEncoder(writer).Encode(Response{responseData})
}

func handleRootPost(writer http.ResponseWriter, request *http.Request) {
	temperature, errTemp := strconv.ParseFloat(request.URL.Query().Get("temperature"), 64)
	humidity, errHumid := strconv.ParseFloat(request.URL.Query().Get("humidity"), 64)
	pressure, errPress := strconv.ParseFloat(request.URL.Query().Get("pressure"), 64)

	if errTemp != nil || errHumid != nil || errPress != nil {
		writer.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(writer).Encode(Response{ErrorResponse{"Invalid data", 400}})
	}

	statement, _ := Database.Prepare("INSERT INTO weather (temperature, humidity, pressure) VALUES (?, ?, ?)")
	_, err := statement.Exec(temperature, humidity, pressure)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(writer).Encode(Response{ErrorResponse{"Internal server error", 500}})
	}

	json.NewEncoder(writer).Encode(Response{WeatherResponse{0, temperature, humidity, pressure}})
}

func handleDevices(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("Request for /devices")
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	switch request.Method {
	case "GET":
		handleDeviceGet(writer, request)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(writer).Encode(Response{ErrorResponse{"Method not allowed", 405}})
	}
}

func handleDeviceGet(writer http.ResponseWriter, request *http.Request) {
	statement, _ := Database.Prepare("SELECT id, name, model FROM devices")
	rows, _ := statement.Query()

	var (
		responseData []DeviceResponse
		id           int
		name         string
		model        string
	)

	for rows.Next() {
		rows.Scan(&id, &name, &model)
		responseData = append(responseData, DeviceResponse{id, name, model})
	}

	json.NewEncoder(writer).Encode(Response{responseData})
}
