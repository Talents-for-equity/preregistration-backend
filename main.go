package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type SibContactResponse struct {
	Contacts []SibContact `json:"contacts"`
	Count    int          `json:"count"`
}

type SibContact struct {
	UpdateEnabled bool       `json:"updateEnabled"`
	Email         string     `json:"email"`
	Attributes    Attributes `json:"attributes"`
}

type Attributes struct {
	RAW_JSON   string `json:"RAW_JSON"`
	NEWSLETTER bool   `json:"NEWSLETTER"`
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
	Avatar     string `json:"avatar"`
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
		url := os.Getenv("SIB_ENDPOINT")
		key := os.Getenv("SIB_KEY")

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("accept", "application/json")
		req.Header.Add("api-key", key)

		res, _ := http.DefaultClient.Do(req)
		defer res.Body.Close()

		sibResponses := SibContactResponse{}
		err := json.NewDecoder(res.Body).Decode(&sibResponses)
		if err != nil {
			log.Fatal(err)
		}

		var preregistrations = []Preregistration{}
		for _, reg := range sibResponses.Contacts {
			if reg.Attributes.RAW_JSON == "" {
				continue
			}
			cont := Preregistration{}
			err := json.NewDecoder(strings.NewReader(reg.Attributes.RAW_JSON)).Decode(&cont)
			if err != nil {
				log.Fatal(err)
			}

			preregistrations = append(preregistrations, Preregistration{
				Name:       "",
				Email:      "",
				Country:    "",
				Zip:        "",
				Linkedin:   "",
				Profession: cont.Profession,
				Talent:     cont.Talent,
				Seeker:     cont.Seeker,
				Newsletter: cont.Newsletter,
				Lon:        cont.Lon,
				Lat:        cont.Lat,
				Avatar:     "",
			})
		}
		out, err := json.Marshal(preregistrations)
		fmt.Fprintf(w, "%s", out)

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

		fmt.Fprintf(w, "%s", "[]")
		key := os.Getenv("SIB_KEY")
		url := os.Getenv("SIB_ENDPOINT")

		rawJson, err := json.Marshal(preregistration)
		if err != nil {
			log.Fatal(err)
		}

		sibContact := SibContact{
			UpdateEnabled: false,
			Email:         preregistration.Email,
			Attributes: Attributes{
				RAW_JSON:   string(rawJson),
				NEWSLETTER: preregistration.Newsletter,
			},
		}
		sibJson, err := json.Marshal(sibContact)
		payload := strings.NewReader(string(sibJson))
		req, _ := http.NewRequest("POST", url, payload)

		req.Header.Add("accept", "application/json")
		req.Header.Add("api-key", key)
		req.Header.Add("content-type", "application/json")

		res, _ := http.DefaultClient.Do(req)

		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)

		fmt.Println(res)
		fmt.Println(string(body))

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func main() {
	http.HandleFunc("/", root)
	http.HandleFunc("/mapping", mapping)

	port := ":8080"
	log.Println("Listening on" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
