package main

import (
	"encoding/json"
	"fmt"
	"gonetdisk/config"
	"gonetdisk/util"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// 本地根目录
var HOME_DIR string
var ARCHIVE_DIR string

// URL路径
const HOME_URL = "/home"

type Nav struct {
	Name   string
	Href   string
	Active bool
}

type RowItem struct {
	Type    string
	Icon    string
	Name    string
	Href    string
	IsDir   bool
	BSize   util.ByteSize
	Size    string
	ModTime string
}

func InitDir(runDir string) {
	logdir := path.Join(runDir, "log")
	if !util.PathExists(logdir) {
		if err := os.MkdirAll(logdir, os.ModePerm); err != nil {
			panic(err)
		}
	}

	logfile, err := os.OpenFile(path.Join(logdir, "logrus.log"), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(logfile)
	}

	log.SetOutput(logfile)

	HOME_DIR = path.Join(runDir, HOME_URL)
	if !util.PathExists(HOME_DIR) {
		if err := os.MkdirAll(HOME_DIR, os.ModePerm); err != nil {
			panic(err)
		}
	}
	ARCHIVE_DIR = path.Join(runDir, "archive")
	if !util.PathExists(ARCHIVE_DIR) {
		if err := os.MkdirAll(ARCHIVE_DIR, os.ModePerm); err != nil {
			panic(err)
		}
	}
}

func ReadDirFromUrlPath(urlpath string, query string) []RowItem {
	localDir := path.Join(HOME_DIR, urlpath)
	fullUrl := path.Join(HOME_URL, urlpath)

	var result []RowItem
	fi, err := ioutil.ReadDir(localDir)
	if err != nil {
		return result
	}

	for i := range fi {
		size := "--"
		bsize := util.ByteSize(fi[i].Size())
		href := path.Join(fullUrl, fi[i].Name())

		isDir := fi[i].IsDir()
		if isDir {
			if len(query) > 0 {
				href += "?" + query
			}
		} else {
			size = bsize.Format()
		}

		name, icon := config.Instance().GetNameAndIcon(path.Join(localDir, fi[i].Name()))
		result = append(result, RowItem{
			Type:    name,
			Icon:    icon,
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

func SortFiles(files []RowItem, s string, o string) []RowItem {
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

	lessString := func(cond bool, left string, right string) bool {
		if cond {
			return left < right
		}
		return left > right
	}

	if s == "name" {
		sort.SliceStable(files, func(i, j int) bool {
			return lessString(isAsc, files[i].Name, files[j].Name)
		})
	} else if s == "time" {
		sort.SliceStable(files, func(i, j int) bool {
			return lessString(isAsc, files[i].ModTime, files[j].ModTime)
		})
	} else if s == "type" {
		sort.SliceStable(files, func(i, j int) bool {
			return lessString(isAsc, files[i].Type, files[j].Type)
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
	if !strings.HasPrefix(url, HOME_URL) {
		return path.Join(HOME_DIR, url)
	}
	return HOME_DIR + url[len(HOME_URL):]
}

func GetRefererPath(c *gin.Context) (string, error) {
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
		if !util.PathExists(pathstr) {
			return pathstr
		}

		ext := filepath.Ext(pathstr)
		pathstr = pathstr[:len(pathstr)-len(ext)] + "_bak" + ext
	}
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

//////////////////////////////////////
type Handler struct {
	app *gin.Engine
}

func NewHandler(engine *gin.Engine) Handler {
	return Handler{
		app: engine,
	}
}

func (handler Handler) RedirectContext(location string, c *gin.Context) {
	c.Request.URL.Path = location
	handler.app.HandleContext(c)
}

func (handler Handler) LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		endTime := time.Now()
		cost := endTime.Sub(startTime)
		method := c.Request.Method
		requrl := c.Request.RequestURI
		log.Info("url:", requrl, ", method:", method, ", cost:", cost.Milliseconds(), "ms")
	}
}

func (handler Handler) Home(c *gin.Context) {
	urlInfo, err := url.Parse(c.Param("path"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	urlPath := urlInfo.Path
	localPath := path.Join(HOME_DIR, urlPath)
	log.Info("url path:", urlPath)
	if util.IsDir(localPath) {
		fullUrl := path.Join(HOME_URL, urlPath)
		RowItemList := ReadDirFromUrlPath(urlPath, c.Request.URL.RawQuery)
		SortFiles(RowItemList, c.Query("s"), c.Query("o"))
		navList := ParseNavList(fullUrl, c.Request.URL.RawQuery)
		data := gin.H{
			"title": urlPath,
			"dir":   urlPath,
			"list":  RowItemList,
			"nav":   navList,
		}
		c.HTML(http.StatusOK, "index.html", data)
		return
	}

	if !util.PathExists(localPath) {
		handler.RedirectContext("/404", c)
		return
	}

	c.FileAttachment(localPath, path.Base(localPath))
}

func (handler Handler) ErrorRender(title string, message string, c *gin.Context) {
	backUrl, err := GetRefererPath(c)
	if err != nil {
		title = "Referer Path Error"
		message = err.Error()
		backUrl = HOME_URL
	}

	c.HTML(http.StatusInternalServerError, "error.html", gin.H{
		"title":   title,
		"message": message,
		"back":    backUrl,
	})
}

func (handler Handler) NoRoute(c *gin.Context) {
	c.HTML(http.StatusOK, "404.html", gin.H{})
}

func (handler Handler) Delete(c *gin.Context) {
	b, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		handler.ErrorRender("Error", err.Error(), c)
		return
	}

	var files []string
	err = json.Unmarshal(b, &files)
	if err != nil {
		handler.ErrorRender("Error", err.Error(), c)
		return
	}

	for _, f := range files {
		localPath := GetLocalPath(f)
		if escapePath, err := url.QueryUnescape(localPath); err == nil {
			localPath = escapePath
		}

		if err = os.RemoveAll(localPath); err != nil {
			log.Error("remove file, name:", f, ", err:", err)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"err": 0,
	})
}

func (handler Handler) New(c *gin.Context) {
	curPath, err := GetRefererPath(c)
	if err != nil {
		return
	}

	name := c.PostForm("name")
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		handler.ErrorRender("Warning", "The name cannot be empty!", c)
		return
	}

	newPath := path.Join(GetLocalPath(curPath), name)
	if err := os.MkdirAll(newPath, os.ModePerm); err != nil {
		handler.ErrorRender("ERROR", err.Error(), c)
		return
	}

	c.Redirect(http.StatusFound, curPath)
}

func (handler Handler) Upload(c *gin.Context) {
	curPath, err := GetRefererPath(c)
	if err != nil {
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		handler.ErrorRender("Warning", err.Error(), c)
		return
	}

	files := form.File["files"]
	dst := GetLocalPath(curPath)
	for _, file := range files {
		dstFile := path.Join(dst, file.Filename)
		dstFile = GetUniquePath(dstFile)
		if err := c.SaveUploadedFile(file, dstFile); err != nil {
			log.Error("save file, name:", file.Filename, ", err:", err)
		}
	}
	c.Redirect(http.StatusFound, curPath)
}

func (handler Handler) Move(c *gin.Context) {
	curpath, err := GetRefererPath(c)
	if err != nil {
		return
	}

	frompath := strings.TrimSpace(c.PostForm("frompath"))
	dstpath := c.PostForm("name")
	if len(frompath) == 0 || len(dstpath) == 0 {
		handler.ErrorRender("Warning", "File path cannot be empty!", c)
		return
	}

	dstpath = GetLocalPath(dstpath)
	dstdir := filepath.Dir(dstpath)
	if !util.PathExists(dstdir) {
		if err := os.MkdirAll(dstdir, os.ModePerm); err != nil {
			handler.ErrorRender("ERROR", err.Error(), c)
			return
		}
	}

	frompath = GetLocalPath(frompath)
	log.Info("remove from:", frompath, ", to:", dstpath)
	if err := os.Rename(frompath, dstpath); err != nil {
		handler.ErrorRender("ERROR", err.Error(), c)
		return
	}

	c.Redirect(http.StatusFound, curpath)
}

func (handler Handler) Archive(c *gin.Context) {
	name := c.PostForm("name")
	if len(name) == 0 {
		handler.ErrorRender("Warning", "The name cannot be empty!", c)
		return
	}

	var pathlist []string
	if err := json.Unmarshal([]byte(c.PostForm("pathlist")), &pathlist); err != nil {
		handler.ErrorRender("ERROR", err.Error(), c)
		return
	}
	log.Info("archive list:", pathlist)

	var zipdir string
	var paramsList []string
	for _, pathstr := range pathlist {
		if escapePath, err := url.QueryUnescape(pathstr); err == nil {
			localpath := GetLocalPath(escapePath)
			if util.PathExists(localpath) {
				name := filepath.Base(localpath)
				if len(zipdir) == 0 {
					zipdir = filepath.Dir(localpath)
				}
				paramsList = append(paramsList, "\""+name+"\"")
			}
		}
	}

	uniqueName, err := util.Uuidv4()
	if err != nil {
		handler.ErrorRender("ERROR UUID", err.Error(), c)
		return
	}

	zippath := path.Join(ARCHIVE_DIR, uniqueName+"_"+name)
	cmdstr := fmt.Sprintf("cd %s && zip -r %s %s", zipdir, zippath, strings.Join(paramsList, " "))
	log.Info("shell command:", cmdstr)
	cmd := exec.Command("/bin/bash", "-c", cmdstr)
	if cmd == nil {
		handler.ErrorRender("ERROR EXEC SHELL", "Exec command shell failed!", c)
		return
	}

	err = cmd.Run()
	if err != nil {
		handler.ErrorRender("ERROR", err.Error(), c)
		return
	}

	c.FileAttachment(zippath, name)
}
