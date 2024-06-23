package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/asticode/go-astisub"
)

type Course struct {
	Id       string
	Author   Author
	Title    string
	Episodes map[string]Episode
	Feed     string
	Updated  string
}

type Author struct {
	Name  string
	Email string
}

type Episode struct {
	Id       string
	Title    string
	Updated  string
	Summary  string
	Files    []string
	Chapters string
}

type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Entry struct {
	Id      string `xml:"id"`
	Updated string `xml:"updated"`
	Title   string `xml:"title"`
	Author  Author `xml:"author"`
	Link    []Link `xml:"link"`
	Summary string `xml:"summary"`
}

type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []Entry  `xml:"entry"`
}

var courses = make(map[string]Course)

// why dont do everything in-place?
// the data structure has duplicates, which forces me to do some sanity checks

func main() {
	var feed Feed
	err := parseFeed("https://rwth.video/courses/feed", &feed)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, entry := range feed.Entries {
		l, err := feedLink(entry.Link)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		var subFeed Feed
		err = parseFeed(l, &subFeed)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		// only save course if the feed is valid
		courses[entry.Id] = Course{
			Id:       entry.Id,
			Author:   entry.Author,
			Title:    entry.Title,
			Feed:     l,
			Updated:  entry.Updated,
			Episodes: make(map[string]Episode),
		}

		fmt.Println("Processing ", entry.Title)

		// unable to append to map fields, so I have to do this sh1t
		course := courses[entry.Id]

		for _, subEntry := range subFeed.Entries {
			// be nice to the server
			time.Sleep(time.Millisecond * 100)
			id, chapters, _ := chapters(subEntry.Link)

			episode, ok := courses[entry.Id].Episodes[id]
			if !ok {
				episode = Episode{
					Id:      subEntry.Id,
					Title:   subEntry.Title,
					Updated: subEntry.Updated,
					Summary: subEntry.Summary,
				}
			}

			if chapters != nil && len(chapters.Items) > 0 {
				buff := new(bytes.Buffer)
				if err = chapters.WriteToSRT(buff); err != nil {
					panic(err)
				}

				episode.Chapters = buff.String()
			}

			for _, l := range subEntry.Link {
				if l.Rel == "enclosure" {
					episode.Files = append(episode.Files, l.Href)
				}
			}

			course.Episodes[id] = episode
		}

		courses[entry.Id] = course
	}

	coursesJSON, err := json.Marshal(courses)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	err = os.WriteFile("courses.json", coursesJSON, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

}

func parseFeed(url string, feed interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return err
	}

	return nil
}

func feedLink(link []Link) (string, error) {
	for _, l := range link {
		if l.Href[len(l.Href)-5:] == "/feed" {
			return l.Href, nil
		}

	}
	return "", fmt.Errorf("feed link not found")
}

func extractNumber(text string) string {
	re := regexp.MustCompile(`\d{4}$`)
	match := re.FindString(text)
	return match
}

func chapters(link []Link) (string, *astisub.Subtitles, error) {
	for _, l := range link {
		match := extractNumber(l.Href)
		if len(match) != 4 {
			continue
		}

		subs, err := parseWebVVT(l.Href)
		if err != nil {
			return "", nil, err
		}

		return match, subs, nil
	}

	return "", nil, fmt.Errorf("link not found")
}

func parseWebVVT(url string) (*astisub.Subtitles, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	subs, err := astisub.ReadFromWebVTT(resp.Body)
	if err != nil {
		return nil, err
	}

	return subs, nil
}

var r, _ = regexp.Compile("\\\\|/|:|\\*|\\?|<|>")

func escape(name string) string {
	return r.ReplaceAllString(name, "")
}
