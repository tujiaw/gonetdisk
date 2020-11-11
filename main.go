package main

import (
	"gonetdisk/config"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func main() {
	InitDir(os.Args[0])
	configPath := path.Join(filepath.Dir(os.Args[0]), "config/config.json")
	if err := config.Instance().Init(configPath); err != nil {
		panic(err)
	}

	log.Info("start home dir:", HOMEDIR)

	app := gin.Default()
	app.LoadHTMLGlob("web/template/*")
	app.Static("/web", "./web")

	app.Use(LoggerHandler())
	app.GET("/home/*path", HomeHandler)
	app.POST("/delete", DeleteHandler)
	app.POST("/new", NewHandler)
	app.POST("/upload", UploadHandler)
	app.POST("/move", MoveHandler)
	app.POST("/archive", ArchiveHandler)

	const PORT = ":8989"
	log.Info("app start listen port", PORT)
	if err := app.Run(PORT); err != nil {
		panic(err)
	}
}
