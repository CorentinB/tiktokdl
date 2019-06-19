package tiktokdl

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/labstack/gommon/color"
	"mvdan.cc/xurls"
)

// Metadata contain video's metadatas
type Metadata struct {
	Username  string `json:"username"`
	PosterURL string `json:"poster_url"`
	VideoURL  string `json:"video_url"`
	Music     string `json:"music"`
	Title     string `json:"title"`
}

var client = http.Client{}

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]")

func init() {
	// Disable HTTP/2: Empty TLSNextProto map
	client.Transport = http.DefaultTransport
	client.Transport.(*http.Transport).TLSNextProto =
		make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
}

func findCDNURL(urls []string) string {
	for _, element := range urls {
		if strings.Contains(element, ".mp4") {
			return element
		}
	}
	return "not found"
}

func downloadFile(url string, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Download take a TikTok URL as parameter and an optional
// output directory, fetch needed informations, and download
// the files.
func Download(url string, outputDirectory string, randomUA, verbose bool) {
	if outputDirectory == "" {
		outputDirectory = "./users/"
	}

	var meta Metadata

	// Create collector
	c := colly.NewCollector()

	// Randomize user agent on every request
	if randomUA == true {
		extensions.RandomUserAgent(c)
	}

	// Find poster link
	c.OnHTML("div._video_card_big_left", func(e *colly.HTMLElement) {
		meta.PosterURL = e.ChildAttr("video", "poster")
	})

	// Find poster link
	c.OnHTML("head", func(e *colly.HTMLElement) {
		CDNurl := findCDNURL(xurls.Strict().FindAllString(e.ChildText("script"), -1))
		if CDNurl != "not found" {
			meta.VideoURL = CDNurl
		}
	})

	// Find music informations
	c.OnHTML("div._video_card_big_meta_info_music", func(e *colly.HTMLElement) {
		meta.Music = e.ChildText("a")
	})

	// Find TikTok's title
	c.OnHTML("h1._video_card_big_meta_info_title", func(e *colly.HTMLElement) {
		meta.Title = e.ChildText("span")
	})

	// Find username
	c.OnHTML("div._video_card_big_user_info_names", func(e *colly.HTMLElement) {
		meta.Username = e.ChildText("p._video_card_big_user_info_nickname")
	})

	c.OnScraped(func(r *colly.Response) {
		if verbose == true {
			fmt.Println(checkPre+" Finished:    ", r.Request.URL)
			fmt.Println(checkPre+" User:        ", meta.Username)
			fmt.Println(checkPre+" Title:       ", meta.Title)
			fmt.Println(checkPre+" Music:       ", meta.Music)
			fmt.Println(checkPre+" Poster link: ", meta.PosterURL)
			fmt.Println(checkPre+" Video link:  ", meta.VideoURL)
			fmt.Println("\n"+checkPre+" Statistics:  ", c.String())
		}

		fileName := outputDirectory + meta.Username + "/" + path.Base(url) + "/" + path.Base(url) + "." + meta.Username + "." + strings.Replace(meta.Title, " ", "_", -1)

		os.MkdirAll(outputDirectory+meta.Username+"/"+path.Base(url), os.ModePerm)

		// Generate and write JSON file
		jsonData, err := json.Marshal(meta)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = ioutil.WriteFile(fileName+".json", jsonData, 0644)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Download poster and video
		err = downloadFile(meta.PosterURL, fileName+".jpg")
		if err != nil {
			panic(err)
		}

		err = downloadFile(meta.VideoURL, fileName+".mp4")
		if err != nil {
			panic(err)
		}
	})

	c.Visit(url)
}
