/*
 * Cryptexly - Copyleft of Xavier D. Johnson.
 * me at xavierdjohnson dot com
 * http://www.xavierdjohnson.net/
 *
 * See LICENSE.
 */
package updater

import (
	"github.com/detroitcybersec/cryptexly/cryptexlyd/events"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/log"
	"net/http"
	"regexp"
	"time"
)

var versionParser = regexp.MustCompile("^https://github\\.com/detroitcybersec/cryptexly/releases/tag/v([\\d\\.a-z]+)$")

func worker(currVersion string) {
	interval := time.Duration(60) * time.Minute
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for {
		log.Debugf("Checking for newer versions ...")

		req, _ := http.NewRequest("GET", "https://github.com/detroitcybersec/cryptexly/releases/latest", nil)
		resp, err := client.Do(req)
		if err != nil {
			if err := events.Setup(); err != nil {
				log.Fatal(err)
			}
			log.Errorf("Error while checking latest version: %s.", err)
			return
		}
		defer resp.Body.Close()

		location := resp.Header.Get("Location")

		log.Debugf("Location header = '%s'", location)

		m := versionParser.FindStringSubmatch(location)
		if len(m) == 2 {
			latest := m[1]
			log.Debugf("Latest version is '%s'", latest)
			if currVersion != latest {
				log.Importantf("Update to %s available at %s.", latest, location)
				events.Add(events.UpdateAvailable(currVersion, latest, location))
			} else {
				log.Debugf("No updates available.")
			}
		} else {
			log.Warningf("Unexpected location header: '%s'.", location)
		}

		time.Sleep(interval)
	}
}

func Start(currVersion string) {
	go worker(currVersion)
}
