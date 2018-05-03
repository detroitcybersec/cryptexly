/*
 * Cryptexly - Copyleft of Xavier D. Johnson.
 * me at xavierdjohnson dot com
 * http://www.xavierdjohnson.net/
 *
 * See LICENSE.
 */
package controllers

import (
	"github.com/detroitcybersec/cryptexly/cryptexlyd/db"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/log"
	"github.com/detroitcybersec/cryptexly/cryptexlyd/utils"
	"github.com/gin-gonic/gin"
)

func ListStores(c *gin.Context) {
	stores, err := db.Stores()
	if err != nil {
		utils.NotFound(c)
	} else {
		log.Api(log.DEBUG, c, "Requested stores.")
		c.JSON(200, stores)
	}
}

func CreateStore(c *gin.Context) {
	var meta db.Meta
	if err := SafeBind(c, &meta); err != nil {
		utils.BadRequest(c)
	} else if store, err := db.Create(&meta); err != nil {
		utils.ServerError(c, err)
	} else {
		log.Api(log.DEBUG, c, "Created store %d.", store.Id)
		c.JSON(200, store)
	}
}

func GetStore(c *gin.Context) {
	store, err := db.GetStore(c.Params.ByName("id"))
	if err != nil {
		utils.NotFound(c)
	} else {
		log.Api(log.DEBUG, c, "Requested store %s.", c.Params.ByName("id"))
		c.JSON(200, store.Meta())
	}
}

func DeleteStore(c *gin.Context) {
	store, err := db.GetStore(c.Params.ByName("id"))
	if err != nil {
		utils.NotFound(c)
	} else if err := db.Delete(store); err != nil {
		utils.ServerError(c, err)
	} else {
		log.Api(log.DEBUG, c, "Deleted store %s.", c.Params.ByName("id"))
		c.JSON(200, gin.H{"msg": "Store deleted."})
	}
}

func UpdateStore(c *gin.Context) {
	store, err := db.GetStore(c.Params.ByName("id"))
	var meta db.Meta
	if err != nil {
		utils.NotFound(c)
	} else if err := SafeBind(c, &meta); err != nil {
		utils.BadRequest(c)
	} else if err := store.Update(&meta); err != nil {
		utils.ServerError(c, err)
	} else {
		log.Api(log.DEBUG, c, "Updated store %s.", c.Params.ByName("id"))
		c.JSON(200, gin.H{"msg": "Store updated."})
	}
}
