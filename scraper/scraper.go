package scraper

import (
	"net"
	"net/http"

	"github.com/chromedp/cdproto/network"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	"github.com/chromedp/chromedp"

	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"pinbackup/board"
	redisClient "pinbackup/redis"
)

var (
	// Regex to find picture count
	regexPinsCount = regexp.MustCompile(`^([0-9,]+)`)
	// Regex to find link to original image
	regexOriginalImageLink = regexp.MustCompile(`(?m)https:\/\/[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,3}/originals\b[-a-zA-Z0-9@:%_\+.~#?&//=]*`)
	downloadQueue          string
)

// Config struct contains all variables needed for scraping a board.
type Config struct {
	RedisHost            string
	RedisPort            int
	LoginName            string
	LoginPassword        string
	SelectorPinsCount    string
	SelectorPreviewPins  string
	BoardsQueue          string
	DownloadQueue        string
	ChromeWsDebuggerHost string
}

// Picture struct contains information after scraping a picture.
// This information is handed over to the downloader.
type Picture struct {
	Url          string
	Host         string
	User         string
	Path         string
	PathSegments []string
}

type scraper struct {
	config *Config
}

// lookupIP resolves a hostname to IPv4 address.
func lookupIP(hostName string) (string, error) {
	ips, err := net.LookupIP(hostName)
	if err != nil {
		return "", err
	}

	if len(ips) == 0 {
		return "", errors.New(fmt.Sprintf("No IP address found for host %s", hostName))
	}

	log.Trace().
		Str("method", "lookupIP").
		Msgf("IP addresses for host %s", hostName)

	// 172.217.1.238
	// 2607:f8b0:4000:80e::200e
	return ips[0].String(), nil
}

// getChromeWsDebugURL connects to IP address where Chrome browser is
// listening on port 9222 to get webservice debug URL.
func (s *scraper) getChromeWsDebugURL() (string, error) {
	var (
		err    error
		config = s.config
	)

	log.Trace().
		Str("method", "getChromeWsDebugURL").
		Msgf("Getting IP address of %s", config.ChromeWsDebuggerHost)

	ip, err := lookupIP(config.ChromeWsDebuggerHost)
	if err != nil {
		return "", err
	}

	// E.g.: ws://127.0.0.1:9222/devtools/browser/2689f9a9-eb4c-4f25-b1cf-f4287088bd46
	log.Debug().
		Str("method", "getChromeWsDebugURL").
		Msgf("ChromeWsDebugIP: %s", ip)

	resp, err := http.Get(fmt.Sprintf("http://%s:9222/json/version", ip))
	if err != nil {
		return "", err
	}

	var result map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result["webSocketDebuggerUrl"].(string), nil
}

// login checks if authentication is already done and tries to login inf not.
func (s *scraper) login(browserCtx context.Context, board *board.Board) error {
	var (
		err           error
		authenticated = false
		config        = s.config

		loginTasks = chromedp.Tasks{
			chromedp.Navigate(fmt.Sprintf("https://" + board.Host + "/login/")),
			chromedp.WaitVisible("#password", chromedp.ByID),
			chromedp.Sleep(10 * time.Second),
			chromedp.SendKeys("#email", config.LoginName, chromedp.ByID),
			chromedp.SendKeys("#password", config.LoginPassword, chromedp.ByID),
			chromedp.Sleep(1 * time.Second),
			chromedp.Click(`.red.SignupButton.active`),
			chromedp.Sleep(5 * time.Second),
		}

		authenticatedTasks = chromedp.Tasks{
			chromedp.ActionFunc(func(ctx context.Context) error {
				cookies, err := network.GetAllCookies().Do(ctx)
				if err != nil {
					return err
				}

				for i, cookie := range cookies {
					log.Trace().
						Str("method", "login").
						Msgf("Cookie: %d / Value: %s", i, cookie.Name)

					if cookie.Name == "_auth" {
						log.Debug().
							Str("method", "login").
							Msgf("Already logged in. Cookie %s available.", cookie.Name)
						authenticated = true
					}
				}

				return nil
			}),
		}
	)

	// Check if we are already authenticated
	err = chromedp.Run(browserCtx, authenticatedTasks)
	if err != nil {
		return err
	}

	if !authenticated {
		log.Debug().
			Str("method", "login").
			Msgf("Login needed.")

		err = chromedp.Run(browserCtx, loginTasks)
		if err != nil {
			return err
		}
	}

	return nil
}

// openURL navigates browser to URL that should be scraped.
func (s *scraper) openURL(browserCtx context.Context, board *board.Board) error {
	var (
		err error
	)

	log.Debug().
		Str("method", "openURL").
		Msgf("Navigating to: https://%s/%s/%s", board.Host, board.User, board.Path)

	err = chromedp.Run(browserCtx, chromedp.Navigate(fmt.Sprintf("https://%s/%s/%s", board.Host, board.User, board.Path)))
	if err != nil {
		return err
	}

	return nil
}

// fetchPinsCount tries to get the number of pins from board page on the
// top left. If no error occurs returns the count otherwise zero and the
// error.
// TODO: Doesn't work very well with boards that contains sections.
// TODO: Needs a timeout in case element with pins count doesn't appear
func (s *scraper) fetchPinsCount(browserCtx context.Context) (int, error) {
	var (
		err          error
		rawPinsCount string
		config       = s.config
	)

	fetchRawPinsCountFromPage := func(rawPinsCount *string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.WaitVisible(config.SelectorPinsCount, chromedp.ByQuery),
			chromedp.Text(config.SelectorPinsCount, rawPinsCount, chromedp.NodeVisible, chromedp.ByQuery),
		}
	}

	// Processes input like "1,291 Pins" and returns a integer. Returns 0 if number
	// couldn't be parsed.
	pinsCountStringToInt := func(pinsCount string) int {
		log.Trace().
			Str("method", "fetchPinsCount").
			Msgf("Pin count (raw): %s", pinsCount)

		// returns something like "91" or "1,291" e.g.
		pinsStr := regexPinsCount.FindString(pinsCount)
		log.Trace().
			Str("method", "fetchPinsCount").
			Msgf("Pin count (regex): %s", pinsStr)

		// remove "," if needed
		pinsStr = strings.Replace(pinsStr, ",", "", -1)
		log.Trace().
			Str("method", "fetchPinsCount").
			Msgf("Pin count (; removed): %s", pinsStr)

		pins, err := strconv.Atoi(pinsStr)
		if err != nil {
			return 0
		}

		return pins
	}

	err = chromedp.Run(browserCtx, fetchRawPinsCountFromPage(&rawPinsCount))
	if err != nil {
		return 0, err
	}

	pinsCount := pinsCountStringToInt(rawPinsCount)

	return pinsCount, nil
}

// scrape parses a board and extracts image URLs. Also scrolls down the
// page until end.
func (s *scraper) scrape(browserCtx context.Context, pinsCount int, board *board.Board) error {
	var (
		err    error
		config = s.config
		pins   = newStringSet() // Set to store preview pins without duplicates
	)

	log.Trace().
		Str("method", "scrape").
		Msgf("Borrow Redis connection from pool.")

	conn, err := redisClient.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Get JavaScript code which "scrolls" page by page so that we can fetch
	// the preview links.
	jsScrollIntoView := renderJsScrollIntoView(config.SelectorPreviewPins)

	log.Trace().
		Str("method", "scrape").
		Msgf("Execute JavaScript to get pins count. Selector: %s", config.SelectorPreviewPins)

	// TODO: Needs timeout
	scrollIntoView := func(res *[]string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.WaitVisible(config.SelectorPreviewPins, chromedp.ByQuery),
			chromedp.Sleep(time.Second * 2),
			chromedp.Evaluate(jsScrollIntoView(), res),
			chromedp.Sleep(time.Second * 2),
		}
	}

	log.Debug().
		Str("method", "scrape").
		Msgf("Page has %s pins.", string(pinsCount))

	// Processes the srcset attribute of a pin. It contains four URLs and we want
	// the "originals" one.
	getOriginalImage := func(srcSetAttr string) string {
		return regexOriginalImageLink.FindString(srcSetAttr)
	}

	log.Trace().
		Str("method", "scrape").
		Msg("Waiting for the first preview pins to appear.")

	if err := chromedp.Run(browserCtx, chromedp.WaitVisible(config.SelectorPinsCount)); err != nil {
		return err
	}

	for {
		// After "scrolling" store the pin links here
		var res []string

		// Before scrolling and scraping next page save current pin count
		pinsSetSizeBeforeScraping := pins.Size()

		log.Debug().
			Str("method", "scrape").
			Msgf("Scraped %d/%d picture links.", pins.Size(), pinsCount)

		// If we have equal or more pins downloaded as posted on the page
		// we stop scraping
		if pins.Size() >= pinsCount {
			break
		}

		log.Trace().
			Str("method", "scrape").
			Msg("Scrolling to next page.")

		// Scroll further down the page and fetch next bunch of pins
		err = chromedp.Run(browserCtx, scrollIntoView(&res))
		if err != nil {
			return err
		}

		// For every pin extract the "/original/" path in srcset
		for _, n := range res {
			link := getOriginalImage(n)
			if !strings.HasPrefix(link, "https") {
				continue
			}
			pins.Add(n)

			log.Trace().
				Str("method", "scrape").
				Msgf("Found picture: %s", link)

			picture := Picture{}
			picture.Url = link
			picture.Host = board.Host
			picture.User = board.User
			picture.Path = board.Path
			picture.PathSegments = board.PathSegments

			message, err := json.Marshal(picture)
			if err != nil {
				return errors.New("Can not decode picture object")
			}

			err = redisClient.Publish(conn, config.DownloadQueue, []byte(message))
			if err != nil {
				return err
			}

		}

		// If we have equal or more pins fetched after scrolling further
		// down the page we stop scraping
		// TODO: Retry 2-3x
		if pinsSetSizeBeforeScraping >= pins.Size() {
			break
		}
	}

	return nil
}

// disablePictureLoading causes the preview images not to load in the browser
// context. That saves quite some memory and makes things faster in general.
// TODO: Needs to be implemented
/*func disablePictureLoading(ctx context.Context) chromedp.Tasks{
	return chromedp.Tasks{
		network.SetRequestInterception([]*network.RequestPattern{
			&network.RequestPattern{
				URLPattern: "*", ResourceType: "Image",
			},
		}),

		chromedp.CallbackFunc("Network.requestIntercepted", func(param interface{}, handler *chromedp.TargetHandler) {
			network.ContinueInterceptedRequest(data.InterceptionID).WithErrorReason("Aborted")).Do(ctx, handler)
		}),
	}
}*/

// StartProcessQueue waits for incoming scrape jobs
func StartProcessQueue(config *Config) error {
	log.Debug().
		Msg("Starting scraper queue.")

	semChan := make(chan bool, 1)

	log.Trace().
		Str("method", "StartProcessQueue").
		Msg("Init Redis connection pool.")

	redisClient.InitPool(config.RedisHost, config.RedisPort)

	log.Trace().
		Str("method", "StartProcessingQueue").
		Msg("Getting Redis connection from pool.")

	conn, err := redisClient.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	// TODO: Needs to be replaced with Redis Streams for persistence.
	psc := redis.PubSubConn{Conn: conn}
	psc.Subscribe(config.BoardsQueue)
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			// Block one slot in buffered channel
			semChan <- true
			go func(message []byte) {
				board := board.Board{}

				decoder := json.NewDecoder(bytes.NewReader(message))
				if err := decoder.Decode(&board); err != nil {
					log.Error().
						Str("method", "StartProcessQueue").
						Msgf("Can not decode board %s. Error: %s", board, err.Error())
					return
				}

				s := scraper{
					config: config,
				}

				chromeWsDebugURL, err := s.getChromeWsDebugURL()
				if err != nil {
					log.Error().
						Str("method", "StartProcessQueue").
						Msgf("Can not get Chrome debug URL: %s", err.Error())
					return
				}

				// Browser context
				bctx, cancelBctx := chromedp.NewRemoteAllocator(context.Background(), chromeWsDebugURL)
				defer cancelBctx()

				// Tab context (create new tab and close afterwards)
				tctx, cancelTctx := chromedp.NewContext(bctx)
				defer cancelTctx()

				s.login(tctx, &board)
				s.openURL(tctx, &board)

				pinsCount, err := s.fetchPinsCount(tctx)
				if err != nil {
					log.Error().
						Str("method", "StartProcessQueue").
						Msgf("Error during fetching pin count: %s", err.Error())
				}

				err = s.scrape(tctx, pinsCount, &board)
				if err != nil {
					log.Error().
						Str("method", "StartProcessQueue").
						Msgf("Error during scraping: %s", err.Error())
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
