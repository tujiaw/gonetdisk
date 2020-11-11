package main

import (
	"gonetdisk/config"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func main() {
	runDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	InitDir(runDir)
	configPath := path.Join(runDir, "config/config.json")
	if err := config.Instance().Init(configPath); err != nil {
		panic(err)
	}

	log.Info("start home dir:", HOME_DIR)

	app := gin.Default()
	handler := NewHandler(app)

	app.HTMLRender = LoadTemplates(path.Join(runDir, "web/template"))
	app.Static("/web", "web")

	app.Use(handler.LoggerMiddleware())
	app.NoRoute(handler.NoRoute)
	app.GET("/home/*path", handler.Home)
	app.GET("/404", handler.NoRoute)
	app.POST("/delete", handler.Delete)
	app.POST("/new", handler.New)
	app.POST("/upload", handler.Upload)
	app.POST("/move", handler.Move)
	app.POST("/archive", handler.Archive)

	const PORT = ":5683"
	log.Info("app start listen port", PORT)
	if err := app.Run(PORT); err != nil {
		panic(err)
	}
}

func LoadTemplates(templatesDir string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	// 非模板嵌套
	htmls, err := filepath.Glob(templatesDir + "/htmls/*.html")
	if err != nil {
		panic(err.Error())
	}
	for _, html := range htmls {
		r.AddFromGlob(filepath.Base(html), html)
	}

	// 布局模板
	layouts, err := filepath.Glob(templatesDir + "/layouts/*.html")
	if err != nil {
		panic(err.Error())
	}

	// 嵌套的内容模板
	includes, err := filepath.Glob(templatesDir + "/includes/*.html")
	if err != nil {
		panic(err.Error())
	}

	// template自定义函数
	funcMap := template.FuncMap{
		"StringToLower": func(str string) string {
			return strings.ToLower(str)
		},
	}

	for _, include := range includes {
		files := []string{}
		files = append(files, templatesDir+"/frame.html", include)
		files = append(files, layouts...)
		r.AddFromFilesFuncs(filepath.Base(include), funcMap, files...)
	}

	return r
}
