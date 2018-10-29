package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"os"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/marni/goigc"
)

type trackDB struct {
	HostURL             string
	DatabaseName        string
	TrackCollectionName string
}

type webhookDB struct {
	HostURL               string
	DatabaseName          string
	WebhookCollectionName string
}

func (db *trackDB) Init() {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

}

func (db *webhookDB) Init() {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

}

func (db *trackDB) Add(s Track) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	err = session.DB(db.DatabaseName).C(db.TrackCollectionName).Insert(s)
	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
	}
}

func (db *webhookDB) Add(s Webhook) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	err = session.DB(db.DatabaseName).C(db.WebhookCollectionName).Insert(s)
	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
	}
}

func (db *trackDB) Count() int {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	count, err := session.DB(db.DatabaseName).C(db.TrackCollectionName).Count()
	if err != nil {
		fmt.Printf("error in Count(): %v", err.Error())
		return -1
	}
	return count

}

func (db *webhookDB) Count() int {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	count, err := session.DB(db.DatabaseName).C(db.WebhookCollectionName).Count()
	if err != nil {
		fmt.Printf("error in Count(): %v", err.Error())
		return -1
	}
	return count

}

func (db *trackDB) Get(keyID string) (Track, bool) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	track := Track{}
	err = session.DB(db.DatabaseName).C(db.TrackCollectionName).Find(bson.M{"id": keyID}).One(&track)
	if err != nil {
		return track, false
	}

	return track, true
}

func (db *webhookDB) Get(keyID string) (Webhook, bool) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	Hook := Webhook{}
	err = session.DB(db.DatabaseName).C(db.WebhookCollectionName).Find(bson.M{"id": keyID}).One(&Hook)
	if err != nil {
		return Hook, false
	}

	return Hook, true
}

func (db *trackDB) Delete() (int, bool) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()
	tracks := session.DB(db.DatabaseName).C(db.TrackCollectionName)

	count, err := tracks.Count()
	if err != nil {
		return 0, false
	}

	tracks.RemoveAll(nil)
	return count, true
}

func (db *webhookDB) Delete(keyID string) bool {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()
	err = session.DB(db.DatabaseName).C(db.WebhookCollectionName).Remove(bson.M{"id": keyID})
	if err != nil {
		return false
	}

	return true
}
// Some .igc files URLs for use in testing
// http://skypolaris.org/wp-content/uploads/IGS%20Files/Madrid%20to%20Jerez.igc
// http://skypolaris.org/wp-content/uploads/IGS%20Files/Jarez%20to%20Senegal.igc
// http://skypolaris.org/wp-content/uploads/IGS%20Files/Boavista%20Medellin.igc
// http://skypolaris.org/wp-content/uploads/IGS%20Files/Medellin%20Guatemala.igc

//MetaInfo struct for easy metainfo use
type MetaInfo struct {
	Uptime  string `json:"uptime"`
	Info    string `json:"info"`
	Version string `json:"version"`
}

//Track stores data about the track
type URLTrack struct {
	ID          string
	HDate       time.Time `json:"H_Date"`
	Pilot       string    `json:"pilot"`
	Glider      string    `json:"glider"`
	GliderID    string    `json:"glider_id"`
	TrackLength float64   `json:"track_length"`
	URL         string    `json:"track_src_url"`
	TimeStamp bson.ObjectId
}

//Ticker stores info used for ticker
type Ticker struct {
	TLatest    bson.ObjectId `json:"t_latest"`
	TStart     bson.ObjectId `json:"t_start"`
	TStop      bson.ObjectId `json:"t_stop"`
	Tracks     []string      `json:"tracks"`
	Processing string 	 `json:"processing"`
}

//Webhook stores webhook info
type Webhook struct {
	ID       string
	URL      string `json:"webhookURL"`
	Value    int    `json:"minTriggerValue"`
	TrackAdd int

}

//WebhookMessage stores data for the webhook to send
type WebhookMessage struct {
	TLatest    bson.ObjectId `json:"t_latest"`
	Tracks     []string      `json:"tracks"`
	Processing string`json:"processing"`
}

type WebHookSend struct {
	Message WebhookMessage `json:"text"`
}

// VARIABLES:
// timestamp when the service started
var timeStarted time.Time
// Keep count of the number of igc files added to the system
var igcCount int
// Map where the igcFiles are in-memory stored
var igcFiles = make(map[string]URLTrack) // map["URL"]urlTrack
// Keep count of the number of webhooks
var webhookAmount int

//var clockSaved int
var deletedTracks int


// makes sure that the same track isn't added twice
func urlInMap(url string) bool {
	for urlInMap := range igcFiles {
		if urlInMap == url {
			return true
		}
	}
	return false
}

func init() {
	igcCount = 0
	webhookAmount = 0
	//clockSaved = 0
	deletedTracks = 0
	timeStarted = time.Now()
}

// ISO8601 duration parsing function
func parseTimeDifference() string {
	var timeDifference int
	result := "P" // Different time intervals are attached to this, if they are != 0
	// Formulas for calculating different time intervals in seconds
	timeLeft := timeDifference
	years := timeDifference / 31557600
	timeLeft -= years * 31557600
	months := timeLeft / 2629800
	timeLeft -= months * 2629800
	weeks := timeLeft / 604800
	timeLeft -= weeks * 604800
	days := timeLeft / 86400
	timeLeft -= days * 86400
	hours := timeLeft / 3600
	timeLeft -= hours * 3600
	minutes := timeLeft / 60
	timeLeft -= minutes * 60
	seconds := timeLeft

	// Add time invervals to the result only if they are different form 0
	if years != 0 {
		result += fmt.Sprintf("%dY", years)
	}
	if months != 0 {
		result += fmt.Sprintf("%dM", months)
	}
	if weeks != 0 {
		result += fmt.Sprintf("%dW", weeks)
	}
	if days != 0 {
		result += fmt.Sprintf("%dD", days)
	}
	if hours != 0 || minutes != 0 || seconds != 0 { // Check in case time intervals are 0
		result += "T"
		if hours != 0 {
			result += fmt.Sprintf("%dH", hours)
		}
		if minutes != 0 {
			result += fmt.Sprintf("%dM", minutes)
		}
		if seconds != 0 {
			result += fmt.Sprintf("%dS", seconds)
		}
	}
	return result
}

//Bad Request error message function as it is used many times
func error400(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

// Calculate the total distance of the track
func calculateTotalDistance(track igc.Track) string {
	totDistance := 0.0
	// For each point of the track, calculate the distance between 2 points in the Point array
	for i := 0; i < len(track.Points)-1; i++ {
		totDistance += track.Points[i].Distance(track.Points[i+1])
	}
	// Parse it to a string value
	return strconv.FormatFloat(totDistance, 'f', 2, 64)
}

func paraglideHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/paragliding/api/", http.StatusSeeOther)
}

func apiHandler(w http.ResponseWriter, r *http.Request) {	
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) != 4 {
		errRouter(w, r)
		return
	}

	upTime := strings.Split(parseTimeDifference(), ".")
	meta := MetaInfo{upTime[0], Info, Version}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(metaJSON)
}

func trackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" { // If method is POST, user has entered the URL
		if r.Body == nil {
			error400(w)
			return
		}

		var data string // POST body is of content-type: JSON; the result can be stored in a map

		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			error400(w)
			return
		}

		track, err := igc.ParseLocation(data) // call the igc library
		if err != nil {
			error400(w)
			return
		}
		igcCount++            // Increase the count
		nID := "igc" + strconv.Itoa(igcCount)

		newTrack := Track{
			ID,
			track.HDate,
			track.Pilot,
			track.GliderType,
			track.GliderID,
			parseTimeDifference(track),
			data,
			bson.NewObjectIdWithTime(time.Now())}

		trackDataBase.Add(newTrack)

		addJSON, err := json.Marshal(newTrack.ID)
		if err != nil {
			error400(w)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(addJSON)

		sendWebhook(w)

	} else if r.Method == "GET" { // If the method is GET
		w.Header().Set("Content-Type", "application/json") // Set response content-type to JSON

		response := []string{}
		for i := deletedTracks + 1; i <= total; i++ {
			tempTrack, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
			if !ok {
				error400(w)
				return
			}
			response = append(response, tempTrack.ID)
		}

		IDJSON, err := json.Marshal(response)
		if err != nil {
			error400(w)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(IDJSON)

	} else {
		error400(w)
	}
		/*for i := igcFiles { // Get all the IDs of .igc files stored in the igcFiles map
			if y != len(igcFiles)-1 { // If it's the last item in the array, don't add the ","
				response += "\"" + igcFiles[j].trackName + "\","
				y++ // Increment the iterator
			} else {
				response += "\"" + igcFiles[j].trackName + "\""
			}
		}
		response += "]"

		fmt.Fprintf(w, response)
	}*/
}

func idHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")
	track := parts[len(parts)-1]
	if track != "" {

		tempTrack, ok := trackDataBase.Get(track)

		if !ok {
			error400(w)
			return
		}

		trackJSON, err := json.Marshal(tempTrack)

		if err != nil {
			error400(w)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(trackJSON)

	} else {
		errRouter(w, r) // If it isn't, send a 404 Not Found status
	}
}

func fieldHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	id := parts[len(parts)-2]
	field := parts[len(parts)-1]

	if id != "" && field != "" {

		tempTrack, ok := trackDataBase.Get(id)

		if !ok {
			error400(w)
		}

		switch field {
		case "pilot":
			Response := tempTrack.Pilot

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "glider":
			Response := tempTrack.Glider

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "glider_id":
			Response := tempTrack.GliderID
			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "track_length":
			fmt.Fprint(w, tempTrack.TrackLength)
		case "H_date":
			Response := tempTrack.HDate.String()

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		case "track_src_url":
			Response := tempTrack.URL

			response, err := errorCheck(Response)
			if err != nil {
				error400(w)
				return
			}
			fmt.Fprint(w, response)
		}
	} else {
		errRouter(w, r)
	}
}

func ticker(w http.ResponseWriter, r *http.Request) {
	processStart := time.Now().UnixNano() / int64(time.Millisecond)
	arrayCap := 5

	var startTime bson.ObjectId
	var stopTime bson.ObjectId

	tracks := []string{}

	for i := deletedTracks + 1; i <= arrayCap + deletedTracks; i++ {
		tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
		if !ok {
			break
		}
		tracks = append(tracks, tempTimeStamp.ID)

		if i == 1 {
			startTime = tempTimeStamp.TimeStamp
		}
		//Defined here many times in case there are less than 5 tracks
		stopTime = tempTimeStamp.TimeStamp
	}

	trackTotal := trackDataBase.Count()
	lastTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(trackTotal))
	if !ok {
		error400(w)
		return
	}

	latest := lastTimeStamp.TimeStamp

	process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
	processString := strconv.FormatInt(process, 10)

	response := Ticker{latest, startTime, stopTime, tracks, processString}

	tickerJSON, err := json.Marshal(response)
	if err != nil {
		error400(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(tickerJSON)
}

func tickerTimeStamp(w http.ResponseWriter, r *http.Request) {
	processStart := time.Now().UnixNano() / int64(time.Millisecond)
	arrayCap := 5

	parts := strings.Split(r.URL.Path, "/")
	stamp := bson.ObjectIdHex(parts[len(parts)-1])

	var startTime bson.ObjectId
	var stopTime bson.ObjectId

	low := 1

	for {
		tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(low))
		if !ok {
			error400(w)
			return
		}

		if stamp > tempTimeStamp.TimeStamp {
			break
		}
		low++
	}

	tracks := []string{}

	for i := low; i <= arrayCap+low; i++ {
		tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
		if !ok {
			break
		}
		tracks = append(tracks, tempTimeStamp.ID)

		if i == 1 {
			startTime = tempTimeStamp.TimeStamp
		}
		//Defined here many times in case there are less than 5 tracks
		stopTime = tempTimeStamp.TimeStamp
	}

	trackTotal := trackDataBase.Count()
	lastTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(trackTotal))
	if !ok {
		error400(w)
		return
	}

	latest := lastTimeStamp.TimeStamp

	process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
	processString := strconv.FormatInt(process, 10)

	response := Ticker{latest, startTime, stopTime, tracks, processString}

	tickerJSON, err := json.Marshal(response)
	if err != nil {
		error400(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(tickerJSON)
}

func newWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		error400(w)
	}

	var newWebhook Webhook

	err := json.NewDecoder(r.Body).Decode(&newWebhook)
	if err != nil {
		error400(w)
		return
	}

	if newWebhook.Value == 0 {
		newWebhook.Value = 1
	}

	webhookAmount++
	newWebhook.ID = strconv.Itoa(webhookAmount)
	newWebhook.TrackAdd = igcCount

	webhookDataBase.Add(newWebhook)

	fmt.Fprintf(w, newWebhook.ID)
}

func sendWebhook(w http.ResponseWriter) {
	processStart := time.Now().UnixNano() / int64(time.Millisecond)

	hooks := webhookDataBase.Count()
	if hooks == 0 {
		return
	}

	for i := deletedTracks + 1; i <= hooks + deletedTracks; i++ {
		tempWH, ok := webhookDataBase.Get(strconv.Itoa(i))
		if !ok {
			error400(w)
			return
		}
		if tempWH.TrackAdd + tempWH.Value == igcCount + deletedTracks {
			tempTimeStamp, ok := trackDataBase.Get("igc" + strconv.Itoa(igcCount))
			if !ok {
				error400(w)
				return
			}
			tracks := []string{}
			for i := tempWH.TrackAdd + 1; i <= igcCount + deletedTracks; i++ {
				track, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
				if !ok {
					error400(w)
					return
				}
				tracks = append(tracks, track.ID)
			}

			process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
			processString := strconv.FormatInt(process, 10)

			messageBody := WebhookMessage{tempTimeStamp.TimeStamp, tracks, processString}

			message := WebHookSend{messageBody}

			messageJSON, err := json.Marshal(message)

			if err != nil {
				error400(w)
				return
			}

			tempWH.TrackAdd = igcCount + deletedTracks
			ok = webhookDataBase.Delete("igc" + strconv.Itoa(i))
			if !ok {
				error400(w)
				return
			}

			webhookDataBase.Add(tempWH)

			resp, err := http.Post(tempWH.URL, "application/json", bytes.NewBuffer(messageJSON))

			var result map[string]interface{}

			json.NewDecoder(resp.Body).Decode(&result)

			log.Println(result)
			log.Println(result["data"])
		}
	}
}

func manageWebhook(w http.ResponseWriter, r *http.Request) {

	parts := strings.Split(r.URL.Path, "/")
	ID := parts[len(parts)-1]

	if r.Method == "GET" {
		tempWH, ok := webhookDataBase.Get(ID)
		if !ok {
			error400(w)
			return
		}

		resp, err := json.Marshal(tempWH)
		if err != nil {
			error400(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	} else if r.Method == "DELETE" {
		tempWH, ok := webhookDataBase.Get(ID)
		if !ok {
			error400(w)
			return
		}

		ok = webhookDataBase.Delete(ID)
		if !ok {
			error400(w)
			return
		}

		resp, err := json.Marshal(tempWH)
		if err != nil {
			error400(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

/*func clockTrigger() {
	if clockSaved < igcCount + deletedTracks {
		processStart := time.Now().UnixNano() / int64(time.Millisecond)

		 tracks := []string{}
		 var tLate bson.ObjectId

		for i := clockSaved + 1; i <= igcCount + deletedTracks; i++ {
			tempTrack, ok := trackDataBase.Get("igc" + strconv.Itoa(i))
			if !ok {
				panic(ok)
			}
			tracks = append(tracks, tempTrack.ID)

			if i == igcCount + deletedTracks {
				tLate = tempTrack.TimeStamp
			}
		}

		process := (time.Now().UnixNano() / int64(time.Millisecond)) - processStart
		processString := strconv.FormatInt(process, 10)
		messageBody := WebhookMessage{tLate,tracks,processString}

		message := WebHookSend{messageBody}

		messageJSON, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}

		//SlackURL is hidden in another file not posted to github to ensure no spam to it
		resp, err := http.Post(SlackURL, "application/json", bytes.NewBuffer(messageJSON))

		var result map[string]interface{}

		json.NewDecoder(resp.Body).Decode(&result)

		log.Println(result)
		log.Println(result["data"])
	}
}*/

func adminGet(w http.ResponseWriter, r *http.Request) {
	count := trackDataBase.Count()
	fmt.Fprint(w, count)
}

func adminDelete(w http.ResponseWriter, r *http.Request) {
	count, ok := trackDataBase.Delete()
	if !ok {
		error400(w)
		return
	}
	fmt.Fprint(w, count)
}

func errRouter(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func main() {
	trackDataBase.Init()
	webhookDataBase.Init()
	router := mux.NewRouter()

	router.HandleFunc("/", errRouter)
	router.HandleFunc("/paragliding/", paraglideHandler)
	router.HandleFunc("/paragliding/api/", apiHandler)
	router.HandleFunc("/paragliding/api/track/", trackHandler)
	router.HandleFunc("/paragliding/api/track/[a-zA-Z0-9]{3,10}/", idHandler)
	router.HandleFunc("/paragliding/api/track/[a-zA-Z0-9]{3,10}/(pilot|glider|glider_id|track_length|H_date)/", fieldHandler)
	router.HandleFunc("/paragliding/api/ticker/latest", tickerLast)
	router.HandleFunc("/paragliding/api/ticker/", ticker)
	router.HandleFunc("/paragliding/api/ticker/{[0-9A-Za-z]}", tickerTimeStamp)
	router.HandleFunc("/paragliding/api/webhook/new_track/", newWebhook)
	router.HandleFunc("/paragliding/api/webhook/new_track/{[0-9A-Za-z]}", manageWebhook)
	router.HandleFunc("/UnexpectedURL/admin/api/tracks_count", adminGet)
	router.HandleFunc("/UnexpectedURL/admin/api/tracks", adminDelete)
	log.Fatal(http.ListenAndServe(":" + os.Getenv("PORT"), nil))
}
