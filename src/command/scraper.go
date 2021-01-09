package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"net/http"
	"regexp"

	"github.com/BKrajancic/FLD-Bot/m/v2/src/service"
	"github.com/BKrajancic/FLD-Bot/m/v2/src/storage"
)

// ScraperConfig is a struct that can be turned into a usable scraper.
type ScraperConfig struct {
	Command       string // Regular expression which triggers this scraper. Can contain capture groups.
	TitleTemplate string // Title template that will be replaced by regex captures (using %s).
	TitleCapture  string // Regex captures for title replacement.
	URL           string // A url to scrape from, can contain one "%s" which is replaced with the first capture group.
	ReplyCapture  string // Regular expression used to parse a webpage.
	Help          string // Help message to display
}

// GetScraperConfigs returns a set of ScraperConfig by reading a file.
// If a file doesn't exist at the given filepath, an example is made in its place,
// and an error is returned.
func GetScraperConfigs(reader io.Reader) ([]ScraperConfig, error) {
	var config []ScraperConfig

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		fmt.Printf("Unable to read buffer")
		return config, nil
	}

	json.Unmarshal(bytes, &config)
	return config, nil
}

func makeExampleScraperConfig(filepath string) error {
	config := []ScraperConfig{}
	bytes, err := json.Marshal(config)

	if err != nil {
		return errors.New("Unable to create example JSON")
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("Unable to create file: %s", filepath)
	}
	defer file.Close()

	_, err = file.Write(bytes)
	if err != nil {
		return fmt.Errorf("Unable to write to file: %s", filepath)
	}
	return fmt.Errorf("File %s did not exist, an example has been writen", filepath)
}

// GetScraper creates a scraper from a config.
func GetScraper(config ScraperConfig) (Command, error) {
	webpageCapture := regexp.MustCompile(config.ReplyCapture)
	titleCapture := regexp.MustCompile(config.TitleCapture)

	curry := func(sender service.Conversation, user service.User, msg [][]string, storage *storage.Storage, sink func(service.Conversation, service.Message)) {
		scraper(config.URL,
			webpageCapture,
			config.TitleTemplate,
			titleCapture,
			sender,
			user,
			msg,
			storage,
			sink,
		)
	}
	regex, err := regexp.Compile(config.Command)
	if err != nil {
		return Command{}, err
	}
	return Command{
		Pattern: regex,
		Exec:    curry,
		Help:    config.Help,
	}, nil
}

// Return the received message
func scraper(urlTemplate string, webpageCapture *regexp.Regexp, titleTemplate string, titleCapture *regexp.Regexp, sender service.Conversation, user service.User, msg [][]string, storage *storage.Storage, sink func(service.Conversation, service.Message)) {
	substitutions := strings.Count(urlTemplate, "%s")
	url := urlTemplate
	if (substitutions > 0) && (msg == nil || len(msg) == 0 || len(msg[0]) < substitutions) {
		sink(sender, service.Message{Description: "An error when building the url."})
		return
	}

	for _, capture := range msg[0][1:] {
		url = fmt.Sprintf(url, capture)
	}

	response, err := http.Get(url)
	if err == nil {
		// Read response data in to memory
		body, err := ioutil.ReadAll(response.Body)

		if err == nil {
			// Create a regular expression to find comments
			bodyS := string(body)

			matches := webpageCapture.FindAllStringSubmatch(bodyS, -1)
			titleMatches := titleCapture.FindAllStringSubmatch(bodyS, -1)

			if matches != nil {
				allCaptures := make([]string, len(matches))
				for i, captures := range matches {
					allCaptures[i] = strings.Join(captures[1:], " ")
				}

				reply := fmt.Sprintf("%s.\n\nRead more at: %s", strings.Join(allCaptures, " "), url)
				replyTitle := titleTemplate

				for _, captures := range titleMatches {
					for _, captureGroup := range captures[1:] {
						replyTitle = fmt.Sprintf(replyTitle, captureGroup)
					}
				}

				sink(sender, service.Message{
					Title:       replyTitle,
					Description: reply,
					URL:         url,
				})
			} else {
				sink(sender, service.Message{Description: "Could not extract data from the webpage."})
			}
		} else {
			sink(sender, service.Message{Description: "An error occurred when processing the webpage."})
		}
	} else {
		sink(sender, service.Message{
			Description: "An error occurred retrieving the webpage.",
			URL:         url,
		})
	}
}
