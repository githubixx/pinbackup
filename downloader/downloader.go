package downloader

import (
	"bytes"
	"encoding/json"
	"fmt"

	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	redisClient "pinbackup/redis"
	"pinbackup/scraper"
)

var (
	// Regex to find filename in request URI
	regexFilename = regexp.MustCompile(`.*/([a-zA-Z0-9].+)`)
)

// Config struct contains all variables needed for scraping a board.
type Config struct {
	RedisHost      string
	RedisPort      int
	StorageType    string
	FsDownloadPath string
	DownloadQueue  string
}

// downloader struct contains configuration for downloader
type downloader struct {
	config *Config
}

// createDirectory creates a directory and all subpaths. All directories
// will be created with permissions specified in fileMode parameter.
func createDirectory(path string, fileMode uint32) error {
	if path == "" {
		return errors.New("Path needed to create destination directory.")
	}

	err := os.MkdirAll(path, os.FileMode(fileMode))

	return err
}

// parseFilename returns the filename that is part of the requestURI e.g.:
// requestURI: /originals/4a/69/a5/4a69a545f70f78cc99b31bb81c49831c.jpg
// Result: 4a69a545f70f78cc99b31bb81c49831c.jpg
func parseFilename(requestURI string) (string, error) {
	if requestURI == "" {
		return "", errors.New("Can not parse filename. RequestURI is empty.")
	}

	filename := regexFilename.FindStringSubmatch(requestURI)

	if filename == nil {
		return "", errors.New("Can not parse filename out of requestURI.")
	}

	return filename[1], nil
}

// download fetches picture and stores it int specified path.
func (d *downloader) download(ctx context.Context, picture *scraper.Picture, fsDownloadPath string) error {
	var (
		err error
	)

	// Parse URL of picture
	parsedURL, err := url.Parse(picture.Url)
	if err != nil {
		return err
	}

	// Parse filename
	filename, err := parseFilename(parsedURL.RequestURI())
	if err != nil {
		return err
	}

	// Construct destination where to store pictures
	destinationPath := fsDownloadPath + "/" + picture.User + "/" + picture.Path
	destinationFile := destinationPath + "/" + filename

	// Skip download if file already exists
	if _, err := os.Stat(destinationFile); !os.IsNotExist(err) {
		log.Trace().
			Str("method", "download").
			Msgf("%s already exists. Skipping...", destinationFile)
		return nil
	}

	// Create destination directory
	err = createDirectory(destinationPath, 0755)
	if err != nil {
		return errors.New(fmt.Sprintf("Creating destination directory for picture %s failed: %s", picture.Url, err.Error()))
	}

	// TODO Implement retry logic
	// TODO Implement timeout
	response, err := http.Get(picture.Url)
	if err != nil {
		return errors.New(fmt.Sprintf("Fetching picture %s failed.", picture.Url))
	}
	defer response.Body.Close()

	// Create destination file
	// TODO Check if filesystem has enough space
	file, err := os.Create(destinationFile)
	if err != nil {
		return errors.New(fmt.Sprintf("Saving picture %s failed: %s", filename, err.Error()))
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return errors.New(fmt.Sprintf("Saving picture %s failed: %s", filename, err.Error()))
	}

	return nil
}

// StartProcessQueue waits for incoming download jobs and download pictures.
func StartProcessQueue(config *Config) error {
	semChan := make(chan bool, 1)

	log.Debug().
		Str("method", "StartProcessQueue").
		Msg("Init Redis connection pool")

	redisClient.InitPool(config.RedisHost, config.RedisPort)
	conn, err := redisClient.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Debug().
		Str("method", "StartProcessingQueue").
		Msg("Getting Redis connection from pool")

	// TODO: Needs to be replaced with Redis Streams for persistence.
	psc := redis.PubSubConn{Conn: conn}
	psc.Subscribe(config.DownloadQueue)
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			// Block one slot in buffered channel
			semChan <- true
			go func(message []byte) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				picture := scraper.Picture{}

				decoder := json.NewDecoder(bytes.NewReader(message))
				if err := decoder.Decode(&picture); err != nil {
					log.Error().
						Str("method", "StartProcessQueue").
						Msgf("Can not decode picture %s. Error: %s", picture, err.Error())
					return
				}

				d := downloader{
					config: config,
				}

				log.Debug().
					Str("method", "StartProcessQueue").
					Msgf("Downloading picture URL: %s", picture.Url)

				err = d.download(ctx, &picture, config.FsDownloadPath)
				if err != nil {
					log.Error().
						Str("method", "StartProcessQueue").
						Msgf("Picture download error: %s", picture.Url)
				}

				// release slot in buffered channel
				<-semChan
			}(v.Data)
		case error:
			// TODO: Needs to be handled
			return v
		}
	}
}
