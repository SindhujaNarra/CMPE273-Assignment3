package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type UberETA struct {
	Request_id      string  `json:"request_id"`
	Status          string  `json:"status"`
	Vehicle         string  `json:"vehicle"`
	Driver          string  `json:"driver"`
	Location        string  `json:"location"`
	ETA             int     `json:"eta"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
}

type PriceStructures struct {
	Prices []PriceStructure `json:"prices"`
}

type Struct_for_put struct {
	trip_route  []string
	trip_visits map[string]int
}

type Final_struct struct {
	theMap map[string]Struct_for_put
}

type PriceStructure struct {
	ProductId       string  `json:"product_id"`
	CurrencyCode    string  `json:"currency_code"`
	DisplayName     string  `json:"display_name"`
	Estimate        string  `json:"estimate"`
	LowEstimate     int     `json:"low_estimate"`
	HighEstimate    int     `json:"high_estimate"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	Duration        int     `json:"duration"`
	Distance        float64 `json:"distance"`
}

type UberResponse struct {
	Cost     int
	Duration int
	Distance float64
}

type InputAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
}

type OutputAddress struct {
	Id      bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	Name    string        `json:"name"`
	Address string        `json:"address"`
	City    string        `json:"city" `
	State   string        `json:"state"`
	Zip     string        `json:"zip"`

	Coordinate struct {
		Lat  string `json:"lat"`
		Lang string `json:"lang"`
	}
}

type TripUpdateResponse struct {
	Id                           bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	Status                       string        `json:"status"`
	Starting_from_location_id    string        `json:"starting_from_location_id"`
	Next_destination_location_id string        `json:"next_destination_location_id"`
	Best_route_location_ids      []string
	Total_uber_costs             int     `json:"total_uber_costs"`
	Total_uber_duration          int     `json:"total_uber_duration"`
	Total_distance               float64 `json:"total_distance"`
	Uber_wait_time_eta           int     `json:"uber_wait_time_eta"`
}

type Internal_data struct {
	Id               string   `json:"_id" bson:"_id,omitempty"`
	Trip_visited     []string `json:"trip_visited"`
	Trip_not_visited []string `json:"trip_not_visited"`
	Trip_completed   int      `json:"trip_completed"`
}

type GoogleCoordinates struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PlaceID string   `json:"place_id"`
		Types   []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

type Response struct {
	Id         bson.ObjectId `json:"id" bson:"_id"`
	Name       string        `json:"name" bson:"name"`
	Address    string        `json:"address" bson:"address"`
	City       string        `json:"city" bson:"city"`
	State      string        `json:"state" bson:"state"`
	Zip        string        `json:"zip" bson:"zip"`
	Coordinate struct {
		Lat string `json:"lat"   bson:"lat"`
		Lng string `json:"lng"   bson:"lng"`
	} `json:"coordinate" bson:"coordinate"`
}

type TripInput struct {
	Starting_from_location_id string `json:"starting_from_location_id"`
	Location_ids              []string
}

type TripOutput struct {
	Id                        bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	Status                    string        `json:"status"`
	Starting_from_location_id string        `json:"starting_from_location_id"`
	Best_route_location_ids   []string
	Total_uber_costs          int     `json:"total_uber_costs"`
	Total_uber_duration       int     `json:"total_uber_duration"`
	Total_distance            float64 `json:"total_distance"`
}

type MongoSession struct {
	session *mgo.Session
}

func newMongoSession(session *mgo.Session) *MongoSession {
	return &MongoSession{session}
}

func (ms MongoSession) GetLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	id := params.ByName("id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	fmt.Print("Before OID")
	oid := bson.ObjectIdHex(id)
	fmt.Print("OID is", oid)

	resp := Response{}

	if err := ms.session.DB("cmpe273").C("locations").FindId(oid).One(&resp); err != nil {
		fmt.Print("Inside fail case")
		w.WriteHeader(404)
		return
	}

	json.NewDecoder(r.Body).Decode(resp)

	mObject, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", mObject)
}

func (ms MongoSession) CreateLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	resp := Response{}

	json.NewDecoder(r.Body).Decode(&resp)

	data := callGoogleAPI(&resp)

	data.Id = bson.NewObjectId()

	ms.session.DB("cmpe273").C("locations").Insert(data)

	mObject, _ := json.Marshal(data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", mObject)
}

func (ms MongoSession) DeleteLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	id := params.ByName("id")

	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	oid := bson.ObjectIdHex(id)
	if err := ms.session.DB("cmpe273").C("locations").RemoveId(oid); err != nil {
		fmt.Print("Inside fail case")
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(200)
}

func (ms MongoSession) UpdateLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	id := params.ByName("id")

	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	oid := bson.ObjectIdHex(id)

	get := Response{}
	put := Response{}

	put.Id = oid

	json.NewDecoder(r.Body).Decode(&put)

	if err := ms.session.DB("cmpe273").C("locations").FindId(oid).One(&get); err != nil {
		w.WriteHeader(404)
		return
	}

	na := get.Name

	object := ms.session.DB("cmpe273").C("locations")

	get = callGoogleAPI(&put)
	object.Update(bson.M{"_id": oid}, bson.M{"$set": bson.M{"address": put.Address, "city": put.City, "state": put.State, "zip": put.Zip, "coordinate": bson.M{"lat": get.Coordinate.Lat, "lng": get.Coordinate.Lng}}})

	get.Name = na

	mObject, _ := json.Marshal(get)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", mObject)

}

func callGoogleAPI(resp *Response) Response {

	address := resp.Address
	city := resp.City

	gstate := strings.Replace(resp.State, " ", "+", -1)
	gaddress := strings.Replace(address, " ", "+", -1)
	gcity := strings.Replace(city, " ", "+", -1)

	uri := "http://maps.google.com/maps/api/geocode/json?address=" + gaddress + "+" + gcity + "+" + gstate + "&sensor=false"

	result, _ := http.Get(uri)

	body, _ := ioutil.ReadAll(result.Body)

	Cords := GoogleCoordinates{}

	err := json.Unmarshal(body, &Cords)
	if err != nil {
		panic(err)
	}

	for _, Sample := range Cords.Results {
		resp.Coordinate.Lat = strconv.FormatFloat(Sample.Geometry.Location.Lat, 'f', 7, 64)
		resp.Coordinate.Lng = strconv.FormatFloat(Sample.Geometry.Location.Lng, 'f', 7, 64)
	}

	return *resp
}

func (ms MongoSession) CreateTrip(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var tripInput TripInput
	var tripOutput TripOutput

	var costs []int
	var duration []int
	var distance []float64
	costFinal := 0
	durationFinal := 0
	distanceFinal := 0.0

	json.NewDecoder(r.Body).Decode(&tripInput)

	starting_id := bson.ObjectIdHex(tripInput.Starting_from_location_id)

	var dbResponse Response
	if err := ms.session.DB("cmpe273").C("locations").FindId(starting_id).One(&dbResponse); err != nil {
		w.WriteHeader(404)
		return
	}

	start_Lat := dbResponse.Coordinate.Lat
	start_Lang := dbResponse.Coordinate.Lng

	for len(tripInput.Location_ids) > 0 {

		for _, loc := range tripInput.Location_ids {

			id := bson.ObjectIdHex(loc)

			var o Response
			if err := ms.session.DB("cmpe273").C("locations").FindId(id).One(&o); err != nil {
				w.WriteHeader(404)
				return
			}
			loc_Lat := o.Coordinate.Lat
			loc_Lang := o.Coordinate.Lng

			getUberResponse := getUberPrice(start_Lat, start_Lang, loc_Lat, loc_Lang)
			fmt.Println("Uber Response is: ", getUberResponse.Cost, getUberResponse.Duration, getUberResponse.Distance)

			costs = append(costs, getUberResponse.Cost)
			duration = append(duration, getUberResponse.Duration)
			distance = append(distance, getUberResponse.Distance)

		}

		fmt.Println("Cost Array", costs)

		min_cost := costs[0]
		var indexNeeded int
		for index, value := range costs {
			if value < min_cost {
				min_cost = value
				indexNeeded = index
			}
		}

		costFinal += min_cost
		durationFinal += duration[indexNeeded]
		distanceFinal += distance[indexNeeded]

		tripOutput.Best_route_location_ids = append(tripOutput.Best_route_location_ids, tripInput.Location_ids[indexNeeded])

		starting_id = bson.ObjectIdHex(tripInput.Location_ids[indexNeeded])
		if err := ms.session.DB("cmpe273").C("locations").FindId(starting_id).One(&dbResponse); err != nil {
			w.WriteHeader(404)
			return
		}
		tripInput.Location_ids = append(tripInput.Location_ids[:indexNeeded], tripInput.Location_ids[indexNeeded+1:]...)

		start_Lat = dbResponse.Coordinate.Lat
		start_Lang = dbResponse.Coordinate.Lng

		costs = costs[:0]
		duration = duration[:0]
		distance = distance[:0]

	}

	Last_loc_id := bson.ObjectIdHex(tripOutput.Best_route_location_ids[len(tripOutput.Best_route_location_ids)-1])
	var o2 Response
	if err := ms.session.DB("cmpe273").C("locations").FindId(Last_loc_id).One(&o2); err != nil {
		w.WriteHeader(404)
		return
	}
	last_loc_Lat := o2.Coordinate.Lat
	last_loc_Lang := o2.Coordinate.Lng

	ending_id := bson.ObjectIdHex(tripInput.Starting_from_location_id)
	var end Response
	if err := ms.session.DB("cmpe273").C("locations").FindId(ending_id).One(&end); err != nil {
		w.WriteHeader(404)
		return
	}
	end_Lat := end.Coordinate.Lat
	end_Lang := end.Coordinate.Lng

	getUberResponse_last := getUberPrice(last_loc_Lat, last_loc_Lang, end_Lat, end_Lang)

	tripOutput.Id = bson.NewObjectId()
	tripOutput.Status = "planning"
	tripOutput.Starting_from_location_id = tripInput.Starting_from_location_id
	tripOutput.Total_uber_costs = costFinal + getUberResponse_last.Cost
	tripOutput.Total_distance = distanceFinal + getUberResponse_last.Distance
	tripOutput.Total_uber_duration = durationFinal + getUberResponse_last.Duration

	ms.session.DB("cmpe273").C("Trips").Insert(tripOutput)

	uj, _ := json.Marshal(tripOutput)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)
}

func (ms MongoSession) GetTrip(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	id := p.ByName("trip_id")
	fmt.Println("Inside get trip", id)

	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	oid := bson.ObjectIdHex(id)
	var tripOutput TripOutput
	if err := ms.session.DB("cmpe273").C("Trips").FindId(oid).One(&tripOutput); err != nil {
		w.WriteHeader(404)
		return
	}

	uj, _ := json.Marshal(tripOutput)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", uj)
}

func getUberPrice(startLat, startLon, endLat, endLon string) UberResponse {
	client := &http.Client{}

	reqURL := fmt.Sprintf("https://sandbox-api.uber.com/v1/estimates/price?start_latitude=%s&start_longitude=%s&end_latitude=%s&end_longitude=%s", startLat, startLon, endLat, endLon)
	fmt.Println("URL formed: " + reqURL)

	req, err := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("Authorization", "Token 9lrTMMq_48AtGaQxawuCY4a89a4jLMoLGSG8P1KD")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error Calling Uber ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Malformed Response: ", err)
	}

	var res PriceStructures
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("Error in unmarshalling: ", err)
	}

	var uberResponse UberResponse
	uberResponse.Cost = res.Prices[0].LowEstimate
	uberResponse.Duration = res.Prices[0].Duration
	uberResponse.Distance = res.Prices[0].Distance

	return uberResponse

}

func (ms MongoSession) UpdateTrip(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	var theStruct Struct_for_put
	var final Final_struct
	final.theMap = make(map[string]Struct_for_put)

	var tripUpdate TripUpdateResponse
	var internal Internal_data

	id := p[0].Value
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	oid := bson.ObjectIdHex(id)
	if err := ms.session.DB("cmpe273").C("Trips").FindId(oid).One(&tripUpdate); err != nil {
		w.WriteHeader(404)
		return
	}

	theStruct.trip_route = tripUpdate.Best_route_location_ids
	theStruct.trip_route = append([]string{tripUpdate.Starting_from_location_id}, theStruct.trip_route...)

	theStruct.trip_visits = make(map[string]int)

	var trip_visited []string
	var trip_not_visited []string

	if err := ms.session.DB("cmpe273").C("Trip_internal_data").FindId(id).One(&internal); err != nil {
		for index, loc := range theStruct.trip_route {
			if index == 0 {

				theStruct.trip_visits[loc] = 1
				trip_visited = append(trip_visited, loc)
			} else {
				theStruct.trip_visits[loc] = 0
				trip_not_visited = append(trip_not_visited, loc)
			}
		}
		internal.Id = id
		internal.Trip_visited = trip_visited
		internal.Trip_not_visited = trip_not_visited
		internal.Trip_completed = 0
		ms.session.DB("cmpe273").C("Trip_internal_data").Insert(internal)

	} else {
		for _, loc_id := range internal.Trip_visited {
			theStruct.trip_visits[loc_id] = 1
		}
		for _, loc_id := range internal.Trip_not_visited {
			theStruct.trip_visits[loc_id] = 0
		}
	}

	fmt.Println("Trip visit map ", theStruct.trip_visits)
	final.theMap[id] = theStruct

	last_index := len(theStruct.trip_route) - 1
	trip_completed := internal.Trip_completed

	if trip_completed == 1 {

		tripUpdate.Status = "completed"

		uj, _ := json.Marshal(tripUpdate)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		fmt.Fprintf(w, "%s", uj)
		return
	}

	for i, location := range theStruct.trip_route {
		if theStruct.trip_visits[location] == 0 {
			tripUpdate.Next_destination_location_id = location
			nextoid := bson.ObjectIdHex(location)
			var o OutputAddress
			if err := ms.session.DB("cmpe273").C("locations").FindId(nextoid).One(&o); err != nil {
				w.WriteHeader(404)
				return
			}
			nlat := o.Coordinate.Lat
			nlang := o.Coordinate.Lang

			if i == 0 {
				starting_point := theStruct.trip_route[last_index]
				startingoid := bson.ObjectIdHex(starting_point)
				var o OutputAddress
				if err := ms.session.DB("cmpe273").C("locations").FindId(startingoid).One(&o); err != nil {
					w.WriteHeader(404)
					return
				}
				slat := o.Coordinate.Lat
				slang := o.Coordinate.Lang

				eta := Get_uber_eta(slat, slang, nlat, nlang)
				tripUpdate.Uber_wait_time_eta = eta
				trip_completed = 1
			} else {
				starting_point2 := theStruct.trip_route[i-1]
				startingoid2 := bson.ObjectIdHex(starting_point2)
				var o OutputAddress
				if err := ms.session.DB("cmpe273").C("locations").FindId(startingoid2).One(&o); err != nil {
					w.WriteHeader(404)
					return
				}
				slat := o.Coordinate.Lat
				slang := o.Coordinate.Lang
				eta := Get_uber_eta(slat, slang, nlat, nlang)
				tripUpdate.Uber_wait_time_eta = eta
			}

			fmt.Println("Starting Location: ", tripUpdate.Starting_from_location_id)
			fmt.Println("Next destination: ", tripUpdate.Next_destination_location_id)
			theStruct.trip_visits[location] = 1
			if i == last_index {
				theStruct.trip_visits[theStruct.trip_route[0]] = 0
			}
			break
		}
	}

	trip_visited = trip_visited[:0]
	trip_not_visited = trip_not_visited[:0]
	for location, visit := range theStruct.trip_visits {
		if visit == 1 {
			trip_visited = append(trip_visited, location)
		} else {
			trip_not_visited = append(trip_not_visited, location)
		}
	}

	internal.Id = id
	internal.Trip_visited = trip_visited
	internal.Trip_not_visited = trip_not_visited
	internal.Trip_completed = trip_completed

	c := ms.session.DB("cmpe273").C("Trip_internal_data")
	id2 := bson.M{"_id": id}
	err := c.Update(id2, internal)
	if err != nil {
		panic(err)
	}

	uj, _ := json.Marshal(tripUpdate)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)

}

func Get_uber_eta(startLat, startLon, endLat, endLon string) int {

	var jsonStr = []byte(`{"start_latitude":"` + startLat + `","start_longitude":"` + startLon + `","end_latitude":"` + endLat + `","end_longitude":"` + endLon + `","product_id":"04a497f5-380d-47f2-bf1b-ad4cfdcb51f2"}`)
	reqURL := "https://sandbox-api.uber.com/v1/requests"
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonStr))

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error in sending req to Uber: ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error in reading response: ", err)
	}

	var res UberETA
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("error in unmashalling response: ", err)
	}
	eta := res.ETA
	return eta

}

func getConnection() *mgo.Session {

	conn, err := mgo.Dial("mongodb://sindhuja:sindhuja@ds057254.mongolab.com:57254/cmpe273")

	if err != nil {
		panic(err)
	}
	return conn
}

func main() {

	r := httprouter.New()

	ms := newMongoSession(getConnection())

	r.GET("/locations/:id", ms.GetLocation)
	r.POST("/locations", ms.CreateLocation)
	r.DELETE("/locations/:id", ms.DeleteLocation)
	r.PUT("/locations/:id", ms.UpdateLocation)

	r.POST("/trips", ms.CreateTrip)
	r.GET("/trips/:trip_id", ms.GetTrip)
	r.PUT("/trips/:trip_id/request", ms.UpdateTrip)

	http.ListenAndServe("localhost:8080", r)

}
