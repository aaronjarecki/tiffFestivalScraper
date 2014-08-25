package main

import (
	"code.google.com/p/go.net/html"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type Movie struct {
	Name        string
	ObjectId    string
	URL         *url.URL
	Schedule    []ScheduleItem
	Pitch       string
	Programme   string
	Description string
}

type ScheduleItem struct {
	DateStr   string
	Date      string
	TimeStr   string
	Venue     string
	VenueRoom string
	QAndA     bool
	Premium   bool
	Press     bool
	School    bool
	Industry  bool
}

type Movies struct {
	Movies []*Movie
}

func (self Movies) String() string {
	str := ""
	for _, item := range self.Movies {
		str = str + fmt.Sprintf("%v", item) + "\n"
	}
	return str
}

var ProgrammeUrls = []string{
	"http://www.tiff.net/festivals/thefestival/programmes/future-projections",
	"http://www.tiff.net/festivals/thefestival/programmes/tiff-docs",
	"http://www.tiff.net/festivals/thefestival/programmes/discovery",
	"http://www.tiff.net/festivals/thefestival/programmes/midnight-madness",
	"http://www.tiff.net/festivals/thefestival/programmes/galapresentations",
	"http://www.tiff.net/festivals/thefestival/programmes/masters",
	"http://www.tiff.net/festivals/thefestival/programmes/specialpresentations",
	"http://www.tiff.net/festivals/thefestival/programmes/mavericks",
	"http://www.tiff.net/festivals/thefestival/programmes/contemporary-world-cinema",
	"http://www.tiff.net/festivals/thefestival/programmes/contemporary-world-speakers",
	"http://www.tiff.net/festivals/thefestival/programmes/wavelengths-all",
	"http://www.tiff.net/festivals/thefestival/programmes/kids",
	"http://www.tiff.net/festivals/thefestival/programmes/city-to-city",
	"http://www.tiff.net/festivals/thefestival/programmes/short-cuts-canada",
	"http://www.tiff.net/festivals/thefestival/programmes/short-cuts-international",
	"http://www.tiff.net/festivals/thefestival/programmes/cinematheque",
	"http://www.tiff.net/festivals/thefestival/programmes/vanguard",
	"http://www.tiff.net/festivals/thefestival/programmes/next-wave",
	"http://www.tiff.net/festivals/thefestival/programmes/special-events",
}

var ProgrammeNames = []string{
	"future-projections",
	"tiff-docs",
	"discovery",
	"midnight-madness",
	"galapresentations",
	"masters",
	"specialpresentations",
	"mavericks",
	"contemporary-world-cinema",
	"contemporary-world-speakers",
	"wavelengths-all",
	"kids",
	"city-to-city",
	"short-cuts-canada",
	"short-cuts-international",
	"cinematheque",
	"vanguard",
	"next-wave",
	"special-events",
}
var CurrentMovie *Movie
var CurrentProgram string
var AllMovies Movies

func main() {
	AllMovies.Movies = make([]*Movie, 0)
	for index, programmeUrl := range ProgrammeUrls {
		log.Printf("<-----Parsing Program: %v----->", programmeUrl)
		CurrentProgram = ProgrammeNames[index]
		ParseProgramme(programmeUrl)
	}
	jsonOutput, err := json.Marshal(&AllMovies)
	if err != nil {
		log.Fatalf("Error marshalling json: %v", err)
	}
	if err := ioutil.WriteFile("/tmp/Tiff2014.json", jsonOutput, 0775); err != nil {
		log.Fatalf("Error writting file: %v", err)
	}
	log.Printf("Success")
}

func ParseProgramme(programmeUrl string) {
	resp, err := http.Get(programmeUrl)
	if err != nil {
		log.Fatalf("Error getting programme: %v", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Error parsing html: %v", err)
	}
	CurrentMovie = new(Movie)
	ParseProgrammeHTML(doc)
}

func ParseMovie(movieUrl *url.URL) {
	time.Sleep(5 * time.Second)
	resp, err := http.Get(movieUrl.String())
	if err != nil {
		log.Fatalf("Error getting movie %v: %v", CurrentMovie.Name, err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Error parsing html: %v", err)
	}
	ParseMovieHTML(doc)
}

func ParseProgrammeHTML(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "h1" {
		h1Node := n.FirstChild
		if !strings.Contains(h1Node.Data, "\n") {
			CurrentMovie.Name = h1Node.Data
		}
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		if url_suffix, ok := LinkIsListItem(n); ok {
			var err error
			CurrentMovie.URL, err = url.Parse("http://www.tiff.net" + url_suffix)
			if err != nil {
				log.Fatalf("Error while trying to parse URL: %v", err)
			}
		}
	}
	if CurrentMovie.Name != "" && CurrentMovie.URL != nil {
		//We've got all we need here, parse the page for the Movie
		log.Printf("Parsing details for %v (%v)", CurrentMovie.Name, CurrentMovie.URL.String())
		ParseMovie(CurrentMovie.URL)
		CurrentMovie.Programme = CurrentProgram
		AllMovies.Movies = append(AllMovies.Movies, CurrentMovie)
		log.Printf("Done with %v (%v)", CurrentMovie.Name, CurrentMovie.ObjectId)
		CurrentMovie = new(Movie)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ParseProgrammeHTML(c)
	}
}

func ParseMovieHTML(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "script" {
		scriptNode := n.FirstChild
		if scriptNode != nil && strings.Contains(scriptNode.Data, "var objectId = '") {
			indexOfObjectId := strings.Index(scriptNode.Data, "objectId") + 12
			CurrentMovie.ObjectId = scriptNode.Data[indexOfObjectId : indexOfObjectId+10]
			GetScreeningSchedule(CurrentMovie.ObjectId)
		}
	}
	if n.Type == html.ElementNode && n.Data == "p" {
		if pitch, ok := ParagraphIsAPitch(n); ok {
			CurrentMovie.Pitch = pitch
		}
		if desc, ok := ParagraphIsADescription(n); ok {
			CurrentMovie.Description = CurrentMovie.Description + "\n" + desc
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ParseMovieHTML(c)
	}
}

func GetScreeningSchedule(objectId string) {
	screeningScheduleUrl := "http://tiff.net//ajax/whats-on-film/" + objectId
	resp, err := http.Get(screeningScheduleUrl)
	if err != nil {
		log.Fatalf("Error getting movie screening schedule %v (%v): %v", CurrentMovie.Name, CurrentMovie.ObjectId, err)
	}
	defer resp.Body.Close()

	jsonSchedule, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading movie screening schedule %v (%v): %v", CurrentMovie.Name, CurrentMovie.ObjectId, err)
	}
	schedule := new(interface{})
	if objectId == "2330049689" {
		log.Printf("The JSON: %v", string(jsonSchedule))
	}
	if strings.Contains(string(jsonSchedule), "whats-on-film") {
		log.Printf("Warning: %v (%v) does not have a schedule", CurrentMovie.Name, CurrentMovie.ObjectId)
	} else {
		if err := json.Unmarshal(jsonSchedule, schedule); err != nil {
			log.Fatalf("Error unmarshalling schedule json: %v", err)
		}
		ParseSchedule(schedule)
	}
}

func ParseSchedule(ptrToSchedule interface{}) {
	// We know that this is a pointer to an interface,
	// so get past the pointer
	valueOfSchedule := reflect.ValueOf(ptrToSchedule).Elem()
	// We now have an interface, so get past that
	valueOfSchedule = valueOfSchedule.Elem()
	// Now our value should be a map

	// Get the schedule array ready
	CurrentMovie.Schedule = make([]ScheduleItem, 0)
	for _, dateKey := range valueOfSchedule.MapKeys() {
		scheduleItem := valueOfSchedule.MapIndex(dateKey).Elem()
		scheduleObject := new(ScheduleItem)

		for _, itemKey := range scheduleItem.MapKeys() {
			if itemKey.String() == "date" {
				scheduleObject.DateStr = scheduleItem.MapIndex(itemKey).Elem().String()
			} else if itemKey.String() == "eventformat" {
				scheduleObject.Date = scheduleItem.MapIndex(itemKey).Elem().String()
			} else if itemKey.String() == "timekeys" {
				//This will be an array of structs
				timeKeys := scheduleItem.MapIndex(itemKey).Elem()
				for i := 0; i < timeKeys.Len(); i++ {
					timeItem := timeKeys.Index(i).Elem()
					for _, itemKey := range timeItem.MapKeys() {
						if itemKey.String() == "starttime" {
							scheduleObject.TimeStr = timeItem.MapIndex(itemKey).Elem().String()
						} else if itemKey.String() == "venue_name" {
							scheduleObject.Venue = timeItem.MapIndex(itemKey).Elem().String()
						} else if itemKey.String() == "room_name" {
							scheduleObject.VenueRoom = timeItem.MapIndex(itemKey).Elem().String()
						} else if itemKey.String() == "extended_q_and_a" {
							if timeItem.MapIndex(itemKey).Elem().String() == "0" {
								scheduleObject.QAndA = false
							} else {
								scheduleObject.QAndA = true
							}
						} else if itemKey.String() == "premium" {
							if timeItem.MapIndex(itemKey).Elem().String() == "0" {
								scheduleObject.Premium = false
							} else {
								scheduleObject.Premium = true
							}
						} else if itemKey.String() == "press" {
							if timeItem.MapIndex(itemKey).Elem().String() == "0" {
								scheduleObject.Press = false
							} else {
								scheduleObject.Press = true
							}
						} else if itemKey.String() == "industry" {
							if timeItem.MapIndex(itemKey).Elem().String() == "0" {
								scheduleObject.Industry = false
							} else {
								scheduleObject.Industry = true
							}
						} else if itemKey.String() == "school" {
							if timeItem.MapIndex(itemKey).Elem().String() == "0" {
								scheduleObject.School = false
							} else {
								scheduleObject.School = true
							}
						}
					}
				}
			}
		}
		CurrentMovie.Schedule = append(CurrentMovie.Schedule, *scheduleObject)
	}
}

func LinkIsListItem(n *html.Node) (string, bool) {
	href := ""
	isListItem := false
	for _, value := range n.Attr {
		if value.Key == "class" && strings.Contains(value.Val, "list-item") {
			isListItem = true
		} else if value.Key == "href" {
			href = value.Val
		}
	}
	return href, isListItem
}

func ParagraphIsAPitch(n *html.Node) (string, bool) {
	for _, value := range n.Attr {
		if value.Key == "class" && strings.Contains(value.Val, "pitch") {
			if n.FirstChild != nil {
				return n.FirstChild.Data, true
			}
		}
	}
	return "", false
}

func ParagraphIsADescription(n *html.Node) (string, bool) {
	parent := n.Parent
	var grandparent *html.Node
	if parent != nil {
		grandparent = parent.Parent
	}
	if grandparent != nil {
		desc := ""
		for _, value := range grandparent.Attr {
			if value.Key == "class" && strings.Contains(value.Val, "film-note") {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						desc = desc + c.Data
					} else if c.Type == html.ElementNode {
						if c.FirstChild != nil {
							desc = desc + c.FirstChild.Data
						}
					}
				}
				return desc, true
			}
		}
	}
	return "", false
}
