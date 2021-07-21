package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	apiKey string
	http.Client
}

type Trip struct {
	StartingPoint string
	Destination   string
	Citizenship   string
	Transit       bool
}

type TripRequirements struct {
	Visa        []map[string]interface{} `json:"visa"`
	Passport    []map[string]interface{} `json:"passport"`
	Vaccination []map[string]interface{} `json:"vaccination"`
}

var baseUrl string

func (c *Client) GetTripRequirements(trip *Trip) (*TripRequirements, error) {
	baseUrl = "https://requirements-api.sandbox.joinsherpa.com/v2/"
	url := fmt.Sprintf(baseUrl+"entry-requirements?citizenship=%s&destination=%s&portOfEntry=%slanguage=en-US&transit=%b", trip.Citizenship, trip.Destination, trip.StartingPoint, trip.Transit)
	var tripData *TripRequirements
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
	c := NewClient("Fake key")

	t := Trip{
		StartingPoint: "RS",
		Destination:   "PRT",
		Citizenship:   "GB",
		Transit:       false,
	}

	trip, err := c.GetTripRequirements(&t)
	if err != nil {
		fmt.Printf("\nReceived error: %v", err)
		return
	}
	fmt.Printf("Get Trip Requirements: %v", trip)
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}
