package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	uuid4 "github.com/google/uuid" // Stupid way to avoid conflict
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
	Address  string
	Version  string
	Location string
}
type DeviceLogin struct {
	Uuid string
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

	err = http.ListenAndServe("0.0.0.0:8080", mux)
	if err != nil {
		log.Fatal(err)
	}
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
	statement, err := database.Prepare("SELECT id, temperature, humidity, pressure FROM weather ORDER BY created_at DESC;")
	if err != nil {
		handleError(writer, request, 500, "Internal server error")
	}
	rows, err := statement.Query()
	if err != nil {
		handleError(writer, request, 500, "Internal server error")
	}

	var (
		responseData                    []WeatherResponse
		id                              int
		temperature, humidity, pressure float64
	)

	for rows.Next() {
		err = rows.Scan(&id, &temperature, &humidity, &pressure)
		if err != nil {
			handleError(writer, request, 500, "Internal server error")
		}
		responseData = append(responseData, WeatherResponse{id, temperature, humidity, pressure})
	}

	err = json.NewEncoder(writer).Encode(Response{responseData})
	if err != nil {
		log.Fatal(err)
	}
}

func handleRootPost(writer http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form")
		handleError(writer, request, 500, "Internal server error")
	}

	uuid := request.FormValue("uuid")

	statement, err := database.Prepare("SELECT id FROM devices WHERE uuid = ? LIMIT 1;")
	row, err := statement.Query(uuid)
	if err != nil {
		fmt.Println("Error checking if device exists")
		handleError(writer, request, 500, "Error checking if device exists")
	}
	if !row.Next() {
		fmt.Println("Device does not exist")
		handleError(writer, request, 403, "You are not real")
	}

	err = statement.Close()
	if err != nil {
		fmt.Println("Error closing statement")
		handleError(writer, request, 500, "Internal server error")
	}
	err = row.Close()
	if err != nil {
		fmt.Println("Error closing row")
		handleError(writer, request, 500, "Internal server error")
	}

	temperature, err := strconv.ParseFloat(request.FormValue("temperature"), 64)
	if err != nil {
		fmt.Println("Error parsing temperature " + request.FormValue("temperature"))
		handleError(writer, request, 500, "Internal server error")
	}

	humidity, err := strconv.ParseFloat(request.FormValue("humidity"), 64)
	if err != nil {
		fmt.Println("Error parsing humidity " + request.FormValue("humidity"))
		handleError(writer, request, 500, "Internal server error")
	}

	pressure, err := strconv.ParseFloat(request.FormValue("pressure"), 64)
	if err != nil {
		fmt.Println("Error parsing pressure " + request.FormValue("pressure"))
		handleError(writer, request, 500, "Internal server error")
	}

	statement, _ = database.Prepare("INSERT INTO weather (temperature, humidity, pressure) VALUES (?, ?, ?);")
	_, err = statement.Exec(temperature, humidity, pressure)
	if err != nil {
		handleError(writer, request, 500, "Internal server error")
	}

	err = statement.Close()
	if err != nil {
		fmt.Println("Error closing statement")
		handleError(writer, request, 500, "Internal server error")
	}
	err = row.Close()
	if err != nil {
		fmt.Println("Error closing row")
		handleError(writer, request, 500, "Internal server error")
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
	case "POST":
		handleDevicePost(writer, request)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
		err = json.NewEncoder(writer).Encode(Response{ErrorResponse{"Method not allowed", 405}})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleDeviceGet(writer http.ResponseWriter, request *http.Request) {
	statement, err := database.Prepare("SELECT id, name, address, version FROM devices;")
	if err != nil {
		handleError(writer, request, 500, "Internal server error")
	}
	rows, err := statement.Query()
	if err != nil {
		handleError(writer, request, 500, "Internal server error")
	}

	var (
		responseData           []DeviceResponse
		id                     int
		name, address, version string
	)

	for rows.Next() {
		err = rows.Scan(&id, &name, &address, &version)
		if err != nil {
			handleError(writer, request, 500, "Internal server error")
			break
		}
		responseData = append(responseData, DeviceResponse{id, name, address, version, "Living room"})
	}

	err = statement.Close()
	if err != nil {
		fmt.Println("Error closing statement")
		handleError(writer, request, 500, "Internal server error")
	}
	err = rows.Close()
	if err != nil {
		fmt.Println("Error closing rows")
		handleError(writer, request, 500, "Internal server error")
	}

	err = json.NewEncoder(writer).Encode(Response{responseData})
	if err != nil {
		log.Fatal(err)
	}
}

func handleDevicePost(writer http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form")
		handleError(writer, request, 500, "Internal server error")
	}

	name := request.FormValue("name")
	version := request.FormValue("version")
	address := request.FormValue("address")

	statement, err := database.Prepare("SELECT uuid FROM devices WHERE address = ? LIMIT 1;")
	row, err := statement.Query(address)
	if err != nil {
		fmt.Println("Error checking if device exists")
		handleError(writer, request, 500, "Error checking if device exists")
	}

	var uuid string

	if row.Next() {
		err = row.Scan(&uuid)
		if err != nil {
			handleError(writer, request, 500, "Internal server error")
		}
	} else {
		uuid = uuid4.NewString()
		statement, _ = database.Prepare("INSERT INTO devices (name, version, address, uuid) VALUES (?, ?, ?, ?);")
		_, err = statement.Exec(name, version, address, uuid)
		if err != nil {
			handleError(writer, request, 500, "Internal server error")
		}
	}

	err = statement.Close()
	if err != nil {
		fmt.Println("Error closing statement")
		handleError(writer, request, 500, "Internal server error")
	}
	err = row.Close()
	if err != nil {
		fmt.Println("Error closing row")
		handleError(writer, request, 500, "Internal server error")
	}

	err = json.NewEncoder(writer).Encode(Response{DeviceLogin{uuid}})
	if err != nil {
		log.Fatal(err)
	}
}

func handleError(writer http.ResponseWriter, request *http.Request, code int, message string) {
	request.Header.Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	err := json.NewEncoder(writer).Encode(Response{ErrorResponse{message, code}})
	if err != nil {
		log.Fatal(err)
	}
}
