package main

import (
	"fmt"
	"gonetdisk/config"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	InitDir(os.Args[0])
	configPath := path.Join(filepath.Dir(os.Args[0]), "config/config.json")
	if err := config.Instance().Init(configPath); err != nil {
		panic(err)
	}

	fmt.Println("home dir:", HOMEDIR)
	app := gin.Default()
	app.LoadHTMLGlob("web/template/*")
	app.Static("/web", "./web")

	app.GET("/home/*path", HomeHandler)
	app.POST("/delete", DeleteHandler)
	app.POST("/new", NewHandler)
	app.POST("/upload", UploadHandler)
	app.POST("/move", MoveHandler)
	app.POST("/archive", ArchiveHandler)

	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
