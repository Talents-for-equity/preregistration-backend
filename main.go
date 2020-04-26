package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type DatabasePreregistration struct {
	ID              string          `json:"id"`
	CreatedOn       string          `json:"created_on"`
	ModifiedOn      string          `json:"modified_on"`
	DisabledOn      interface{}     `json:"disabled_on"`
	RegistrationRaw Preregistration `json:"registration_raw"`
}

type Preregistration struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Country    string `json:"country"`
	Zip        string `json:"zip"`
	Linkedin   string `json:"linkedin"`
	Profession string `json:"profession"`
	Talent     bool   `json:"talent"`
	Seeker     bool   `json:"seeker"`
	Newsletter bool   `json:"newsletter"`
	Lon        string `json:"lon"`
	Lat        string `json:"lat"`
}

type Nominatim []struct {
	PlaceID     int      `json:"place_id"`
	Licence     string   `json:"licence"`
	OsmType     string   `json:"osm_type"`
	OsmID       int      `json:"osm_id"`
	Boundingbox []string `json:"boundingbox"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	DisplayName string   `json:"display_name"`
	Class       string   `json:"class"`
	Type        string   `json:"type"`
	Importance  float64  `json:"importance"`
}

func root(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	if err != nil {
		log.Fatal("Fatal error in root", err)
	}
}

func nominatimRequest(address string) Nominatim {
	nom := Nominatim{}
	req, err := http.NewRequest("GET", "https://nominatim.openstreetmap.org/search", nil)
	if err != nil {
		log.Fatal("geohashing error", err)
		return nom
	}

	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("q", address)
	req.URL.RawQuery = q.Encode()
	resp, err := http.Get(req.URL.String())
	if err != nil {
		log.Fatal("geohashing error", err)
		return nom
	}
	err = json.NewDecoder(resp.Body).Decode(&nom)
	if err != nil {
		return nom
	}
	return nom
}

func mapping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	switch r.Method {
	case "OPTIONS":
		return
	case "GET":
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))

	case "POST":
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}

		var preregistration Preregistration
		err := json.NewDecoder(r.Body).Decode(&preregistration)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		nom := nominatimRequest(preregistration.Country + " " + preregistration.Zip)
		if len(nom) == 0 {
			fmt.Fprintf(w, "%s", "[]")
			return
		}
		preregistration.Lat = nom[0].Lat
		preregistration.Lon = nom[0].Lon
		//out, err := json.Marshal(preregistration)
		databasePreregistration := DatabasePreregistration{
			RegistrationRaw: preregistration,
		}

		status := databaseRequest(databasePreregistration)
		fmt.Fprintf(w, "%+v", string(status))

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func databaseRequest(preregistration DatabasePreregistration) (status int) {
	url := os.Getenv("DB_ADDRESS")
	key := os.Getenv("DB_KEY")

	preregistrationStr, err := json.Marshal(preregistration)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(preregistrationStr))
	req.Header.Set("Authorization", key)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	return resp.StatusCode
}

func main() {
	http.HandleFunc("/", root)
	http.HandleFunc("/mapping", mapping)

	port := ":8080"
	log.Println("Listening on" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
