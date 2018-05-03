/*
 * Cryptexly - Copyleft of Xavier D. Johnson.
 * me at xavierdjohnson dot com
 * http://www.xavierdjohnson.net/
 *
 * See LICENSE.
 */
package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/detroitcybersec/cryptexly/cryptexlyd/app"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/backup"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/config"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/controllers"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/db"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/events"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/log"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/middlewares"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/scheduler"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/tls"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/updater"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/utils"

	"github.com/gin-gonic/gin"
)

var (
	signals        = make(chan os.Signal, 1)
	appPath        = ""
	confFile       = ""
	debug          = false
	logfile        = ""
	noColors       = false
	noAuth         = false
	noUpdates      = false
	export         = false
	importFrom     = ""
	output         = "cryptexly.tar"
	dbIsNew        = false
	tlsFingerprint = ""
	router         = (*gin.Engine)(nil)
)

func init() {
	flag.StringVar(&appPath, "app", ".", "Path of the web application to serve.")
	flag.StringVar(&confFile, "config", "", "JSON configuration file.")
	flag.BoolVar(&noAuth, "no-auth", noAuth, "Disable authentication.")
	flag.BoolVar(&noUpdates, "no-updates", noUpdates, "Disable updates check.")

	flag.BoolVar(&debug, "log-debug", debug, "Enable debug logs.")
	flag.StringVar(&logfile, "log-file", logfile, "Log messages to this file instead of standard error.")
	flag.BoolVar(&noColors, "log-colors-off", noColors, "Disable colored output.")

	flag.StringVar(&importFrom, "import", importFrom, "Import stores from this TAR export file.")
	flag.BoolVar(&export, "export", export, "Export store to a TAR archive, requires --output parameter.")
	flag.StringVar(&output, "output", output, "Export file name.")
}

func arcSignalHandler() {
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	s := <-signals
	log.Raw("\n")
	log.Importantf("RECEIVED SIGNAL: %s", s)
	db.Flush()
	os.Exit(1)
}

func setupLogging() {
	var err error

	log.WithColors = !noColors

	if logfile != "" {
		log.Output, err = os.Create(logfile)
		if err != nil {
			log.Fatal(err)
		}

		defer log.Output.Close()
	}

	if debug == true {
		log.MinLevel = log.DEBUG
	} else {
		log.MinLevel = log.INFO
	}
}

func setupDatabase() {
	var err error

	if dbIsNew, err = db.Setup(); err != nil {
		log.Fatal(err)
	}

	if export == true {
		started := time.Now()
		if err = db.Export(output); err != nil {
			log.Fatal(err)
		}
		log.Infof("Archived %s of data in %s to %s.", utils.FormatBytes(db.Size), time.Since(started), log.Bold(output))
		os.Exit(0)
	} else if importFrom != "" {
		started := time.Now()
		if err = db.Import(importFrom); err != nil {
			log.Fatal(err)
		}
		log.Infof("Imported %s of data in %s.", utils.FormatBytes(db.Size), time.Since(started))
		os.Exit(0)
	}
}

func setupScheduler() {
	if config.Conf.Scheduler.Enabled {
		if err := events.Setup(); err != nil {
			log.Fatal(err)
		}

		log.Debugf("Starting scheduler with a period of %ds ...", config.Conf.Scheduler.Period)
		scheduler.Start(config.Conf.Scheduler.Period)
	} else {
		log.Importantf("Scheduler is disabled.")
	}
}

func setupBackups() {
	if config.Conf.Backups.Enabled {
		log.Debugf("Starting backup task with a period of %ds ...", config.Conf.Backups.Period)
		backup.Start(config.Conf.Backups.Period, config.Conf.Backups.Folder, config.Conf.Backups.Run)
	} else {
		log.Importantf("Backups are disabled.")
	}
}

func setupUpdates() {
	if noUpdates == false {
		updater.Start(config.APP_VERSION)
	}
}

func setupTLS() {
	var err error

	if config.Conf.Certificate, err = utils.ExpandPath(config.Conf.Certificate); err != nil {
		log.Fatal(err)
	} else if config.Conf.Key, err = utils.ExpandPath(config.Conf.Key); err != nil {
		log.Fatal(err)
	}

	if utils.Exists(config.Conf.Certificate) == false || utils.Exists(config.Conf.Key) == false {
		log.Importantf("TLS certificate files not found, generating new ones ...")
		if err = tls.Generate(&config.Conf); err != nil {
			log.Fatal(err)
		}
		log.Infof("New RSA key and certificate have been generated, remember to add them as exceptions to your browser!")
	}

	tlsFingerprint, err = tls.Fingerprint(config.Conf.Certificate)
	if err != nil {
		log.Fatal(err)
	}

	log.Importantf("TLS certificate fingerprint is %s", log.Bold(tlsFingerprint))
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router = gin.New()

	err, webapp := app.Open(appPath)
	if err != nil {
		log.Fatal(err)
	}

	router.Use(middlewares.Security(tlsFingerprint))
	router.Use(middlewares.ServeStatic("/", webapp.Path, webapp.Manifest.Index))

	api := router.Group("/api")
	router.POST("/auth", controllers.Auth)

	if noAuth == false {
		api.Use(middlewares.AuthHandler())
	} else {
		log.Importantf("API authentication is disabled.")
	}

	controllers.App = webapp

	api.GET("/status", controllers.GetStatus)
	api.GET("/manifest", controllers.GetManifest)
	api.GET("/config", controllers.GetConfig)

	api.GET("/events/clear", controllers.ClearEvents)

	api.GET("/stores", controllers.ListStores)
	api.POST("/stores", controllers.CreateStore)
	api.GET("/store/:id", controllers.GetStore)
	api.PUT("/store/:id", controllers.UpdateStore)
	api.DELETE("/store/:id", controllers.DeleteStore)

	api.GET("/store/:id/records", controllers.ListRecords)
	api.POST("/store/:id/records", controllers.CreateRecord)
	api.GET("/store/:id/record/:r_id", controllers.GetRecord)
	api.GET("/store/:id/record/:r_id/buffer", controllers.GetRecordBuffer)
	api.PUT("/store/:id/record/:r_id", controllers.UpdateRecord)
	api.DELETE("/store/:id/record/:r_id", controllers.DeleteRecord)

	return router
}

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "password" {
		password := os.Args[2]
		cost := bcrypt.DefaultCost
		if len(os.Args) == 4 {
			n, err := strconv.Atoi(os.Args[3])
			if err != nil {
				log.Fatal(err)
			}
			cost = n
		}
		fmt.Println(config.Conf.HashPassword(password, cost))
		return
	}

	flag.Parse()

	go arcSignalHandler()

	setupLogging()

	log.Infof("%s (%s %s) is starting ...", log.Bold(config.APP_NAME+" v"+config.APP_VERSION), runtime.GOOS, runtime.GOARCH)
	if confFile != "" {
		if err := config.Load(confFile); err != nil {
			log.Fatal(err)
		}
	}

	setupDatabase()
	setupScheduler()
	setupBackups()
	setupUpdates()
	setupTLS()
	setupRouter()

	address := fmt.Sprintf("%s:%d", config.Conf.Address, config.Conf.Port)
	if address[0] == ':' {
		address = "0.0.0.0" + address
	}

	log.Infof("Running on %s ...", log.Bold("https://"+address+"/"))
	if err := router.RunTLS(address, config.Conf.Certificate, config.Conf.Key); err != nil {
		log.Fatal(err)
	}
}
