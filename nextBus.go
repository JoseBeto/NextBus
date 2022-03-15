package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type Route struct {
	Route_id    string
	Agency_id   int
	Route_label string
}

type RouteDirection struct {
	Direction_id   int
	Direction_name string
}

type PlaceCode struct {
	Place_code  string
	Description string
}

type RouteDepartures struct {
	Departures []Departure
}

type Departure struct {
	Departure_time int64
}

var routes []Route

func main() {
	busRoute, busStop, direction, errMsg := parseArgs()
	if errMsg != "" {
		fmt.Println(errMsg)
		return
	}
	fmt.Println(calculateTimeTillNextBus(busRoute, busStop, direction))
}

func parseArgs() (string, string, string, string) {
	if len(os.Args) != 4 {
		return "", "", "", "Not enough arguments. Use: go run nextBus.go [BusRoute] [BusStop] [Direction]"
	}
	return os.Args[1], os.Args[2], os.Args[3], ""
}

func calculateTimeTillNextBus(busRoute string, busStop string, direction string) string {
	// Get Bus routes
	err := getRoutes()
	if err != nil {
		return "Error retrieving routes: " + err.Error()
	}

	// Loop through routes and retrieve route that the user has requested
	var selectedRoute Route
	for _, route := range routes {
		if route.Route_label == busRoute {
			selectedRoute = route
			break
		}
	}

	// Return error if route not found
	if selectedRoute.Route_label == "" {
		return "Error: Route not found"
	}

	direction_id, err := getBusDirectionID(selectedRoute.Route_id, direction)
	if err != nil {
		return "Error getting bus direction ID: " + err.Error()
	}

	place_code, err := getBusStopPlaceCode(selectedRoute.Route_id, direction_id, busStop)
	if err != nil {
		return "Error getting bus direction ID: " + err.Error()
	}

	timeTillNextBusStop, err := getTimeTillNextBusStop(selectedRoute.Route_id, direction_id, place_code)
	if err != nil {
		return "Error getting time till next bus stop: " + err.Error()
	}

	return timeTillNextBusStop
}

/**
This function retrieves bus routes
*/
func getRoutes() error {
	resp, err := http.Get("https://svc.metrotransit.org/nextripv2/routes")
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &routes)
	if err != nil {
		return err
	}
	return nil
}

/**
This function returns a routes direction_id
Params:
	route_id: 		int
	direction:		string
Returns:
	direction_id:	int
	err:			error
*/
func getBusDirectionID(route_id string, direction string) (direction_id int, err error) {
	resp, err := http.Get(fmt.Sprintf("https://svc.metrotransit.org/nextripv2/directions/%s", route_id))
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var routeDirections []RouteDirection
	err = json.Unmarshal(body, &routeDirections)
	if err != nil {
		return
	}

	for _, routeDirection := range routeDirections {
		if strings.Contains(strings.ToLower(routeDirection.Direction_name), direction) {
			return routeDirection.Direction_id, nil
		}
	}
	return direction_id, errors.New("Route direction not found")
}

/**
This function get a bus stop's place code
Params:
	route_id		string
	direction_id	int
	busStop			string
Returns:
	place_code		string
	err				error
*/
func getBusStopPlaceCode(route_id string, direction_id int, busStop string) (place_code string, err error) {
	resp, err := http.Get(fmt.Sprintf("https://svc.metrotransit.org/nextripv2/stops/%s/%d", route_id, direction_id))
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var placeCodes []PlaceCode
	err = json.Unmarshal(body, &placeCodes)
	if err != nil {
		return
	}

	for _, placeCode := range placeCodes {
		if strings.Contains(placeCode.Description, busStop) {
			return placeCode.Place_code, nil
		}
	}
	return place_code, errors.New("Bus stop place code not found")
}

/**
This function returns time till next bus stop and error (if any).
Params:
	route_id			string
	direction_id		int
	place_code			string
Retruns:
	timeTillNextBusStop	string
	err					error
*/
func getTimeTillNextBusStop(route_id string, direction_id int, place_code string) (timeTillNextBusStop string, err error) {
	resp, err := http.Get(fmt.Sprintf("https://svc.metrotransit.org/nextripv2/%s/%d/%s", route_id, direction_id, place_code))
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var routeDepartures RouteDepartures
	err = json.Unmarshal(body, &routeDepartures)
	if err != nil {
		return
	}

	if len(routeDepartures.Departures) == 0 {
		return "", err
	}

	departure_time := time.Unix(routeDepartures.Departures[0].Departure_time, 0)
	currentTime := time.Now()
	diff := departure_time.Sub(currentTime)

	return fmt.Sprintf("%d Minutes", int(diff.Minutes())), nil
}