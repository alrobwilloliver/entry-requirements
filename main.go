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

type Client struct {
	apiKey string
	http.Client
}

type Trip struct {
	StartingPoint string `json:"from"`
	Destination   string `json:"to"`
	Date          time.Time
	Result        TripInfo
}

type TripInfo struct {
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

var baseUrl string
var TripStruct Trip
var tripData TripInfo

func (c *Client) GetTripRequirements(trip Trip) (*TripInfo, error) {
	baseUrl = "https://sandbox.travelperk.com/travelsafe/restrictions"
	url := fmt.Sprintf(baseUrl+"?destination=%s&destination_type=country_code&origin=%s&origin_type=country_code&date=2020-10-15", trip.Destination, trip.StartingPoint)
	err := c.Get(url, &tripData)
	if err != nil {
		return nil, err
	}

	return &tripData, nil
}

func (c *Client) Get(url string, result interface{}) (err error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}

	return c.doRequest(req, result)
}

func (c *Client) doRequest(req *http.Request, result interface{}) (err error) {
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

func main() {
	c := NewClient(os.Getenv("entryApi"))
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static", fs))
	mux.HandleFunc("/", c.handler)
	mux.HandleFunc("/searchEntry", c.searchEntry)
	err := http.ListenAndServe(":3000", http.TimeoutHandler(mux, 5*time.Second, "Timed Out"))

	if err != nil {
		fmt.Printf("\nReceived error: %v", err)
		return
	}
}

func (c *Client) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Context-Type", "application/x-www-form-urlencoded")

	entryFormTemplate, err := template.New("layout.html").Funcs(template.FuncMap{"capital": strings.Title}).ParseGlob("templates/*html")
	if err != nil {
		panic(err)
	}

	err = entryFormTemplate.ExecuteTemplate(w, "layout", nil)
	if err != nil {
		panic(err)
	}
}

func (c *Client) searchEntry(w http.ResponseWriter, r *http.Request) {

	entryFormTemplate, err := template.New("layout.html").Funcs(template.FuncMap{"capital": strings.Title}).ParseGlob("templates/*html")
	if err != nil {
		panic(err)
	}
	var from string
	var to string
	from = r.URL.Query().Get("from")
	to = r.URL.Query().Get("to")
	TripStruct.Destination = to
	TripStruct.StartingPoint = from

	trip, err := c.GetTripRequirements(TripStruct)

	if err != nil {
		fmt.Printf("\nReceived error: %v", err.Error())
		_ = entryFormTemplate.ExecuteTemplate(w, "error", err.Error())
		return
	}
	TripStruct.Result = *trip

	err = entryFormTemplate.ExecuteTemplate(w, "layout", TripStruct)
	if err != nil {
		fmt.Printf("err %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}
