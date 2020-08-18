package server

import (
	"net/http"
	"strings"

	"github.com/WOo0W/bowerbird/config"
	"github.com/WOo0W/bowerbird/helper"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/mongo"
)

func errHandler(err error, c echo.Context) {
	he, ok := err.(*echo.HTTPError)
	if ok && he.Code == http.StatusNotFound &&
		c.Request().Method == http.MethodGet {
		if strings.HasPrefix(c.Request().URL.Path, "/api") {
			c.Echo().DefaultHTTPErrorHandler(err, c)
		} else {
			if err := c.File("../bowerbird-ui/build/index.html"); err != nil {
				c.Echo().DefaultHTTPErrorHandler(err, c)
			}
		}
	} else {
		c.Echo().DefaultHTTPErrorHandler(err, c)
	}
}

// Serve runs a new bowerbird server with the given config
func Serve(conf *config.Config, db *mongo.Database) error {
	e := echo.New()
	e.Debug = true

	e.Use(middleware.GzipWithConfig(
		middleware.GzipConfig{
			Skipper: func(c echo.Context) bool {
				return strings.HasPrefix(c.Request().URL.Path, "/media")
			},
			Level: -1,
		},
	))
	e.Use(middleware.BodyLimit("1M"))

	e.Static("/", "../bowerbird-ui/build")

	pxtr := &http.Transport{}
	err := helper.SetTransportProxy(pxtr, conf.Pixiv.DownloaderProxy, conf.Network.GlobalProxy)
	if err != nil {
		return err
	}
	h := &handler{
		db:             db,
		conf:           conf,
		clientPximg:    &http.Client{Transport: pxtr},
		parsedPixivDir: conf.Storage.ParsedPixiv(),
	}
	e.GET("/api", h.apiVersion)
	e.Static("/api/v1/local/pixiv", conf.Storage.ParsedPixiv())
	e.GET("/api/v1/media/by-id/:id", h.mediaByID)
	e.POST("/api/v1/db/find/:collection", h.dbFind)
	e.GET("/api/v1/proxy/*", h.proxy)

	e.HTTPErrorHandler = errHandler
	return e.Start(conf.Server.Address)
}