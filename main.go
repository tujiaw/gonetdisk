package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// 本地根目录
const HOMEDIR = "/Users/ningto/project/gonetdisk/home"

// URL路径
const HOMEURL = "/home"

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

func GetLocalPath(url string) string {
	if !strings.HasPrefix(url, HOMEURL) {
		return path.Join(HOMEDIR, url)
	}
	return HOMEDIR + url[len(HOMEURL):]
}

func GetCurrentPath(c *gin.Context) (string, error) {
	referer := c.Request.Referer()
	urlInfo, err := url.Parse(referer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return "", err
	}
	return urlInfo.Path, nil
}

func GetUniquePath(pathstr string) string {
	for {
		if !PathExists(pathstr) {
			return pathstr
		}

		ext := filepath.Ext(pathstr)
		pathstr = pathstr[:len(pathstr)-len(ext)] + "_bak" + ext
	}
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	log.Fatal("path exist, path:", path, "err:", err)
	return false
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

func main() {
	app := gin.Default()
	app.MaxMultipartMemory = 8 << 20 // 8MB
	app.LoadHTMLGlob("web/template/*")
	app.Static("/web", "./web")
	app.GET("/home/*path", func(c *gin.Context) {
		relativePath := c.Param("path")
		absolutePath := path.Join(HOMEDIR, c.Param("path"))
		fmt.Println("relative path:", relativePath)
		if IsDir(absolutePath) {
			navPath := path.Join(HOMEURL, relativePath)
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

	app.POST("/delete", func(c *gin.Context) {
		fmt.Println("delete")
		b, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(500, gin.H{
				"err": err.Error(),
			})
			return
		}

		var files []string
		err = json.Unmarshal(b, &files)
		if err != nil {
			c.JSON(500, gin.H{
				"err": err.Error(),
			})
			return
		}

		for _, f := range files {
			localPath := GetLocalPath(f)
			if escapePath, err := url.QueryUnescape(localPath); err == nil {
				localPath = escapePath
			}
			fmt.Println("will delete:", localPath)
			if err = os.RemoveAll(localPath); err != nil {
				log.Fatal("remove file, f:", f, ", err:", err)
			}
		}
		c.JSON(200, gin.H{
			"err": 0,
		})
	})

	app.POST("/new", func(c *gin.Context) {
		curPath, err := GetCurrentPath(c)
		if err != nil {
			return
		}

		name := c.PostForm("name")
		fmt.Println("new name:", name)
		name = strings.TrimSpace(name)
		if len(name) == 0 {
			c.JSON(http.StatusOK, gin.H{"err": "name is empty"})
			return
		}

		newPath := path.Join(GetLocalPath(curPath), name)
		fmt.Println("new folder:", newPath)
		if err := os.MkdirAll(newPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
			return
		}
		fmt.Println("redirect:", curPath)
		c.Redirect(http.StatusFound, curPath)
	})

	app.POST("upload", func(c *gin.Context) {
		curPath, err := GetCurrentPath(c)
		if err != nil {
			return
		}

		form, err := c.MultipartForm()
		if err != nil {
			log.Fatal("form error", err)
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
			return
		}

		files := form.File["files"]
		dst := GetLocalPath(curPath)
		for _, file := range files {
			dstFile := path.Join(dst, file.Filename)
			dstFile = GetUniquePath(dstFile)
			if err := c.SaveUploadedFile(file, dstFile); err != nil {
				log.Fatal("save error, name:", file.Filename, ", err:", err)
			}
		}
		c.Redirect(http.StatusFound, curPath)
	})

	app.POST("/move", func(c *gin.Context) {
		curpath, err := GetCurrentPath(c)
		if err != nil {
			return
		}

		frompath := strings.TrimSpace(c.PostForm("frompath"))
		dstpath := c.PostForm("name")
		if len(frompath) == 0 || len(dstpath) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"err": "from path is empty"})
			return
		}

		dstpath = GetLocalPath(dstpath)
		dstdir := filepath.Dir(dstpath)
		if !PathExists(dstdir) {
			if err := os.MkdirAll(dstdir, os.ModePerm); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
				return
			}
		}

		frompath = GetLocalPath(frompath)
		fmt.Println("from path:", frompath)
		fmt.Println("dst path:", dstpath)
		if err := os.Rename(frompath, dstpath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
			return
		}

		c.Redirect(http.StatusFound, curpath)
	})

	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
