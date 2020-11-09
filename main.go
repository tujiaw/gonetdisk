package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// 本地根目录
var HOMEDIR string
var ARCHIVEDIR string

// URL路径
const HOMEURL = "/home"

type Nav struct {
	Name   string
	Href   string
	Active bool
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

type Item struct {
	Type    string
	Name    string
	Href    string
	IsDir   bool
	BSize   ByteSize
	Size    string
	ModTime string
}

func Uuidv4() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return "", err
	}
	return fmt.Sprintf("%X%X%X%X%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func GetFileType(pathstr string) string {
	if IsDir(pathstr) {
		return "文件夹"
	}

	ext := filepath.Ext(pathstr)
	switch ext {
	case ".exe":
		return "应用程序"
	case ".dll":
		return "应用程序扩展"
	case ".bat":
		return "Windows批处理文件"
	default:
		return "文件"
	}
}

func ReadDir(dir string, root string, query string) []Item {
	var result []Item
	fmt.Println("read dir:", dir)
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalln(err)
		return result
	}
	for i := range fi {
		size := "--"
		bsize := ByteSize(fi[i].Size())
		href := path.Join(root, fi[i].Name())

		isDir := fi[i].IsDir()
		if isDir {
			if len(query) > 0 {
				href += "?" + query
			}
		} else {
			size = bsize.Format()
		}

		result = append(result, Item{
			Type:    GetFileType(fi[i].Name()),
			Name:    fi[i].Name(),
			Href:    href,
			IsDir:   isDir,
			BSize:   bsize,
			Size:    size,
			ModTime: fi[i].ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	return result
}

func SortFiles(files []Item, s string, o string) []Item {
	if len(s) == 0 {
		return files
	}

	var isAsc bool
	if o == "asc" {
		isAsc = true
	} else if o == "desc" {
		isAsc = false
	} else {
		return files
	}

	if s == "name" {
		sort.SliceStable(files, func(i, j int) bool {
			if isAsc {
				return files[i].Name < files[j].Name
			} else {
				return files[i].Name > files[j].Name
			}
		})
	} else if s == "time" {
		sort.SliceStable(files, func(i, j int) bool {
			if isAsc {
				return files[i].ModTime < files[j].ModTime
			} else {
				return files[i].ModTime > files[j].ModTime
			}
		})
	} else if s == "type" {
		sort.SliceStable(files, func(i, j int) bool {
			if isAsc {
				return files[i].Type < files[j].Type
			} else {
				return files[i].Type > files[j].Type
			}
		})
	} else if s == "size" {
		sort.SliceStable(files, func(i, j int) bool {
			if isAsc {
				return files[i].BSize < files[j].BSize
			} else {
				return files[i].BSize > files[j].BSize
			}
		})
	}
	return files
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

func ParseNavList(navpath string, query string) []Nav {
	var result []Nav
	var href string
	nameList := strings.Split(navpath, "/")
	for i, name := range nameList {
		if len(name) > 0 {
			href += "/" + name
			curHref := href
			if len(query) > 0 {
				curHref += "?" + query
			}
			result = append(result, Nav{
				Name:   name,
				Href:   curHref,
				Active: i == len(nameList)-1,
			})
		}
	}
	return result
}

func main() {
	runDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	HOMEDIR = path.Join(runDir, HOMEURL)
	if !PathExists(HOMEDIR) {
		if err := os.MkdirAll(HOMEDIR, os.ModePerm); err != nil {
			panic(err)
		}
	}
	ARCHIVEDIR = path.Join(runDir, "archive")
	if !PathExists(ARCHIVEDIR) {
		if err := os.MkdirAll(ARCHIVEDIR, os.ModePerm); err != nil {
			panic(err)
		}
	}

	fmt.Println("home dir:", HOMEDIR)
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
			itemList := ReadDir(absolutePath, navPath, c.Request.URL.RawQuery)
			SortFiles(itemList, c.Query("s"), c.Query("o"))
			navList := ParseNavList(navPath, c.Request.URL.RawQuery)
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

	app.POST("/upload", func(c *gin.Context) {
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

	app.POST("/archive", func(c *gin.Context) {
		name := c.PostForm("name")
		if len(name) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"err": "name is empty!"})
			return
		}

		var pathlist []string
		if err := json.Unmarshal([]byte(c.PostForm("pathlist")), &pathlist); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
			return
		}
		fmt.Println("path list:", pathlist)

		var paramsList []string
		for _, pathstr := range pathlist {
			if escapePath, err := url.QueryUnescape(pathstr); err == nil {
				localpath := GetLocalPath(escapePath)
				if PathExists(localpath) {
					paramsList = append(paramsList, "\""+localpath+"\"")
				}
			}
		}

		uniqueName, err := Uuidv4()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "uuid error"})
			return
		}

		zippath := path.Join(ARCHIVEDIR, name+uniqueName)
		cmd := exec.Command("zip", zippath, strings.Join(paramsList, " "))
		if cmd == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "cmd is error"})
			return
		}

		err = cmd.Run()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
			return
		}

		c.FileAttachment(zippath, name)
	})

	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
