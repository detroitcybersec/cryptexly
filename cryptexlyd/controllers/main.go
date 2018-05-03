/*
 * Cryptexly - Copyleft of Xavier D. Johnson.
 * me at xavierdjohnson dot com
 * http://www.xavierdjohnson.net/
 *
 * See LICENSE.
 */
// Cryptexly is an API server for your secrets.
//
//     Schemes: http, https
//     BasePath: /
//     Version: 1.0.0
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package controllers

import (
	"github.com/detroitcybersec/cryptexly/cryptexlyd/app"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/config"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/db"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/events"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/log"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gin-gonic/gin/json"
	"io"
	"time"
)

type Status struct {
	Online  bool            `json:"online"`
	Started time.Time       `json:"started"`
	Version string          `json:"version"`
	Size    *uint64         `json:"size"`
	Events  *[]events.Event `json:"events"`
}

var App *app.App
var ServerStatus = Status{
	Online:  true,
	Started: time.Now(),
	Version: config.APP_VERSION,
	Size:    &db.Size,
	Events:  &events.Pool,
}

func SafeBind(c *gin.Context, obj interface{}) error {
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, config.Conf.MaxReqSize))
	if binding.EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if err := decoder.Decode(obj); err != nil {
		return err
	}

	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(obj)
}

// swagger:route GET /api/status status getStatus
//
// Handler returning the current server status.
//
// Produces:
//     - application/json
//
// Responses:
//        200: Status
func GetStatus(c *gin.Context) {
	// log.Api(log.DEBUG, c, "Requested status.")
	c.JSON(200, ServerStatus)
}

func ClearEvents(c *gin.Context) {
	events.Clear()
	c.JSON(200, ServerStatus)
}

// swagger:route GET /api/manifest manifest getManifest
//
// Handler returning the current web application manifest.
//
// Produces:
//     - application/json
//
// Responses:
//        200: Manifest
func GetManifest(c *gin.Context) {
	log.Api(log.DEBUG, c, "Requested manifest.")
	c.JSON(200, App.Manifest)
}

// swagger:route GET /api/config configuration getConfig
//
// Handler returning the current server configuration.
//
// Produces:
//     - application/json
//
// Responses:
//        200: Configuration
func GetConfig(c *gin.Context) {
	log.Api(log.DEBUG, c, "Requested configuration.")
	c.JSON(200, config.Conf)
}
