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

var entryFormTemplate *template.Template
var resultTemplate *template.Template

type Client struct {
	apiKey string
	http.Client
}

type Trip struct {
	StartingPoint string
	Destination   string
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

func Capitalize(s string) string {
	return strings.Title(strings.ToLower(s))
}

func HandleDate(t time.Time) string {
	newTime := t.String()
	time, _ := time.Parse(newTime, newTime)
	return time.Format("2021-11-01")
}

func init() {
	var err error
	entryFormTemplate, err = template.ParseFiles("entry.html")
	if err != nil {
		panic(err)
	}
}

func (c *Client) GetTripRequirements(trip *Trip) (*TripInfo, error) {
	baseUrl = "https://sandbox.travelperk.com/travelsafe/restrictions"
	url := fmt.Sprintf(baseUrl+"?destination=%s&destination_type=country_code&origin=%s&origin_type=country_code&date=2020-10-15", trip.StartingPoint, trip.Destination)
	var tripData *TripInfo
	err := c.Get(url, &tripData)
	if err != nil {
		return nil, err
	}

	return tripData, nil
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

	http.HandleFunc("/", c.handler)
	http.HandleFunc("/searchEntry", c.searchEntry)
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		fmt.Printf("\nReceived error: %v", err)
		return
	}
}

func (c *Client) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Context-Type", "text/html")

	t := Trip{}

	err := entryFormTemplate.Execute(w, t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (c *Client) searchEntry(w http.ResponseWriter, r *http.Request) {

	t := Trip{
		StartingPoint: r.FormValue("from"),
		Destination:   r.FormValue("to"),
		Date:          time.Now(),
	}

	trip, err := c.GetTripRequirements(&t)

	if err != nil {
		fmt.Printf("\nReceived error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Result = *trip

	resultTemplate, err = template.ParseFiles("result.html")
	if err != nil {
		panic(err)
	}

	input := `<h1>Origin: {{.Result.Origin.Name}} Destination: {{.Result.Destination.Name}}</h1>
    <h2>{{capitalize .Result.AuthorizationStatus}}</h2>
	<p>{{.Result.Summary}}. {{.Result.Details}} as of {{.Result.StartDate}}.<p>
	{{range .Result.Requirements}}
		<p>{{.Summary}}.</p>
		{{range .Documents}}
			<p>Fill in documents prior to arrival: </p>
			<a href="{{.document_url}}">{{.document_url}}</a>
		{{end}}
	{{end}}
    <a href="/">Back to search</a>`

	fmap := template.FuncMap{
		"capitalize": Capitalize,
		"handleDate": HandleDate,
	}
	resultTemplate, err = resultTemplate.Funcs(fmap).Parse(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	resultTemplate.Execute(w, t)
	fmt.Printf("Get Trip Requirements: %v", trip)
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}
