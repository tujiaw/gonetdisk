package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// 根目录
const HOMEDIR = "/Users/ningto/project/gonetdisk"

type Nav struct {
	Name   string
	Href   string
	Active bool
}

type Item struct {
	Name    string
	Href    string
	IsDir   bool
	Size    string
	ModTime string
}

type ByteSize float64

const (
	_           = iota             // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota) // 1 << (10*1)
	MB                             // 1 << (10*2)
	GB                             // 1 << (10*3)
	TB                             // 1 << (10*4)
	PB                             // 1 << (10*5)
	EB                             // 1 << (10*6)
	ZB                             // 1 << (10*7)
	YB                             // 1 << (10*8)
)

func (b ByteSize) Format() string {
	if b >= GB {
		return fmt.Sprintf("%.2f GB", b/GB)
	} else if b >= MB {
		return fmt.Sprintf("%.2f MB", b/MB)
	} else if b >= KB {
		return fmt.Sprintf("%.2f KB", b/KB)
	} else {
		return fmt.Sprintf("%v B", b)
	}
}

func ReadDir(dir string, root string) []Item {
	var result []Item
	fmt.Println("read dir:", dir)
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalln(err)
		return result
	}
	for i := range fi {
		size := "--"
		if !fi[i].IsDir() {
			size = ByteSize(fi[i].Size()).Format()
		}
		result = append(result, Item{
			Name:    fi[i].Name(),
			Href:    path.Join(root, fi[i].Name()),
			IsDir:   fi[i].IsDir(),
			Size:    size,
			ModTime: fi[i].ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	return result
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func ParseNavList(navpath string) []Nav {
	var result []Nav
	var href string
	nameList := strings.Split(navpath, "/")
	for i, name := range nameList {
		if len(name) > 0 {
			href += "/" + name
			result = append(result, Nav{
				Name:   name,
				Href:   href,
				Active: i == len(nameList)-1,
			})
		}
	}
	return result
}

func FormatSize(size int64) string {
	return fmt.Sprintf("%v KB", size)
}

func main() {
	app := gin.Default()
	app.LoadHTMLGlob("web/template/*")
	app.Static("/web", "./web")
	app.GET("/home/*path", func(c *gin.Context) {
		relativePath := c.Param("path")
		absolutePath := path.Join(HOMEDIR, c.Param("path"))
		fmt.Println("relative path:", relativePath)
		if IsDir(absolutePath) {
			navPath := path.Join("/home", relativePath)
			itemList := ReadDir(absolutePath, navPath)
			navList := ParseNavList(navPath)
			c.HTML(http.StatusOK, "frame.html", gin.H{
				"title": relativePath,
				"dir":   relativePath,
				"list":  itemList,
				"nav":   navList,
			})
			return
		}

		fmt.Println("absolute path:", absolutePath)
		fmt.Println("base:", path.Base(absolutePath))
		c.FileAttachment(absolutePath, path.Base(absolutePath))
	})

	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
