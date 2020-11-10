package main

import (
	"encoding/json"
	"fmt"
	"gonetdisk/config"
	"gonetdisk/util"
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

type Item struct {
	Type    string
	Icon    string
	Name    string
	Href    string
	IsDir   bool
	BSize   util.ByteSize
	Size    string
	ModTime string
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
		bsize := util.ByteSize(fi[i].Size())
		href := path.Join(root, fi[i].Name())

		isDir := fi[i].IsDir()
		if isDir {
			if len(query) > 0 {
				href += "?" + query
			}
		} else {
			size = bsize.Format()
		}

		name, icon := config.Instance().GetNameAndIcon(path.Join(dir, fi[i].Name()))
		result = append(result, Item{
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

func InitDir(arg0 string) {
	runDir, err := filepath.Abs(filepath.Dir(arg0))
	if err != nil {
		panic(err)
	}
	HOMEDIR = path.Join(runDir, HOMEURL)
	if !util.PathExists(HOMEDIR) {
		if err := os.MkdirAll(HOMEDIR, os.ModePerm); err != nil {
			panic(err)
		}
	}
	ARCHIVEDIR = path.Join(runDir, "archive")
	if !util.PathExists(ARCHIVEDIR) {
		if err := os.MkdirAll(ARCHIVEDIR, os.ModePerm); err != nil {
			panic(err)
		}
	}
}

func HomeHandler(c *gin.Context) {
	relativePath := c.Param("path")
	absolutePath := path.Join(HOMEDIR, c.Param("path"))
	fmt.Println("relative path:", relativePath)
	if util.IsDir(absolutePath) {
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
}

func DeleteHandler(c *gin.Context) {
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
}

func NewHandler(c *gin.Context) {
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
}

func UploadHandler(c *gin.Context) {
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
}

func MoveHandler(c *gin.Context) {
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
	if !util.PathExists(dstdir) {
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
}

func ArchiveHandler(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, gin.H{"err": "uuid error"})
		return
	}

	zippath := path.Join(ARCHIVEDIR, uniqueName+"_"+name)
	cmdstr := fmt.Sprintf("cd %s && zip -r %s %s", zipdir, zippath, strings.Join(paramsList, " "))
	fmt.Println("cmd str:", cmdstr)
	cmd := exec.Command("/bin/bash", "-c", cmdstr)
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
}
