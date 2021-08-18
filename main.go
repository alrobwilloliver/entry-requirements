package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// this is the structure of all app data including the result of the TravelPerk API and the user form inputs
type trip struct {
	StartingPoint string `json:"from"`
	Destination   string `json:"to"`
	Date          time.Time
	Result        tripInfo
}

// the structure of the result from the TravelPerk dummy data
type tripInfo struct {
	Origin struct {
		Name        string `json:"name"`
		CountryCode string `json:"country_code"`
		Type        string `json:"type"`
	} `json:"origin"`
	Destination struct {
		Name        string `json:"name"`
		CountryCode string `json:"country_code"`
		Type        string `json:"type"`
	} `json:"destination"`
	AuthorizationStatus string    `json:"authorization_status"`
	Summary             string    `json:"summary"`
	Details             string    `json:"details"`
	StartDate           string    `json:"start_date"`
	EndDate             string    `json:"end_date"`
	UpdatedAt           time.Time `json:"updated_at"`
	Requirements        []struct {
		Category struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"category"`
		SubCategory struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"sub_category"`
		Summary   string        `json:"summary"`
		Details   string        `json:"details"`
		StartDate string        `json:"start_date"`
		EndDate   string        `json:"end_date"`
		Documents []interface{} `json:"documents"`
	} `json:"requirements"`
}

// the base url to call the TravelPerk API
const baseUrl string = "https://sandbox.travelperk.com/travelsafe/restrictions"

// the client will contain the TravelPerk API key and be an instance of an http.Client
type Client struct {
	apiKey string
	http.Client
}

// the handler function is called on the / route in the front end which renders the front end template
func (c *Client) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")

	entryFormTemplate, err := template.New("layout.html").Funcs(template.FuncMap{"capital": strings.Title}).ParseGlob("templates/*html")
	if err != nil {
		_ = entryFormTemplate.ExecuteTemplate(w, "error", "Failed to load the page!")
		return
	}

	err = entryFormTemplate.ExecuteTemplate(w, "layout", nil)
	if err != nil {
		_ = entryFormTemplate.ExecuteTemplate(w, "error", "Failed to load the page!")
		return
	}
}

// this function is called when the user hits the /searchEntry route by hitting the front end form search button to search for COVID travel info based on the user input. It will also render that information onto the page.
func (c *Client) searchEntry(w http.ResponseWriter, r *http.Request) {

	entryFormTemplate, err := template.New("layout.html").Funcs(template.FuncMap{"capital": strings.Title}).ParseGlob("templates/*html")
	if err != nil {
		_ = entryFormTemplate.ExecuteTemplate(w, "error", "Failed to load the page!")
		return
	}
	var from string
	var to string
	// getting the form data from the user input
	from = r.URL.Query().Get("from")
	to = r.URL.Query().Get("to")
	// this is the structure of all app data including the result of the TravelPerk API and the user form inputs
	var TripStruct trip
	TripStruct.Destination = to
	TripStruct.StartingPoint = from

	trip, err := c.getTripRequirements(TripStruct)

	if err != nil {
		_ = entryFormTemplate.ExecuteTemplate(w, "error", "Failed to access the Covid page data!")
		return
	}
	TripStruct.Result = *trip

	err = entryFormTemplate.ExecuteTemplate(w, "layout", TripStruct)
	if err != nil {
		fmt.Printf("err %v", err)
		_ = entryFormTemplate.ExecuteTemplate(w, "error", "The Covid Page Data Failed to Load!")
	}
}

// this function will call the TravelPerk API
func (c *Client) getTripRequirements(trip trip) (*tripInfo, error) {

	// generate the url with data from the form in the Trip struct to call the TravelPerk API
	url := fmt.Sprintf(baseUrl+"?destination=%s&destination_type=country_code&origin=%s&origin_type=country_code&date=2020-10-15", trip.Destination, trip.StartingPoint)
	// create a struct to inject the TravelPerk API data into
	var tripData tripInfo

	// call the TravelPerk API
	err := c.doRequest(url, &tripData)
	if err != nil {
		return nil, err
	}

	return &tripData, nil
}

// this function will handle the request and response from the TravelPerk API
func (c *Client) doRequest(url string, result interface{}) (err error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Api-Version", "1")
	req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	resp, err := c.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			err = fmt.Errorf(resp.Status)
			return
		}
		err = fmt.Errorf("%s: %s", resp.Status, body)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	return err
}

// this main function starts the whole application
func main() {
	// we create a new client with the TravelPerk API Key from an environment variable
	c := newClient(os.Getenv("ENTRYAPI"))
	// we get the port environment variable supplied by Heroku
	port := os.Getenv("PORT")
	// we create a new Server using Mux
	mux := http.NewServeMux()
	// the file server will serve all the files in the ./static folder (images and css)
	fs := http.FileServer(http.Dir("./static"))
	// this will handle any route on /static as a root /
	mux.Handle("/static/", http.StripPrefix("/static", fs))
	// the mux server will use the client functions to determine behaviour on / and /searchEntry routes
	mux.HandleFunc("/", c.handler)
	mux.HandleFunc("/searchEntry", c.searchEntry)
	// the configured server is ready to serve on the port with a timeout on any route of 5 seconds
	err := http.ListenAndServe(":"+port, http.TimeoutHandler(mux, 5*time.Second, "Timed Out"))

	if err != nil {
		fmt.Printf("\nReceived error: %v", err)
		return
	}
}

// this will create a new Client which will allow the HTTP requests to be made as well as the functions on the HandleFunc routes
func newClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}
