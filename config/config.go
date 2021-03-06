package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tujiaw/goutil"
)

type JsonInfo struct {
	NameIcon []struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	} `json:"nameicon"`

	ExtName []struct {
		Ext  string `json:"ext"`
		Name string `json:"name"`
	} `json:"extname"`

	Preview struct {
		Limit int64    `json:"limit"`
		List  []string `json:"list"`
	} `json:"preview"`

	Address string `json:"address"`
	Passwd  string `json:"passwd"`
}

type Info struct {
	extName      map[string]string
	nameIcon     map[string]string
	previewLimit int64
	previewList  []string
	address      string
	passwd       string
}

var instance *Info
var once sync.Once

func Instance() *Info {
	once.Do(func() {
		instance = &Info{}
		instance.extName = make(map[string]string)
		instance.nameIcon = make(map[string]string)
	})
	return instance
}

func (info *Info) Init(pathstr string) error {
	b, err := ioutil.ReadFile(pathstr)
	if err != nil {
		return err
	}

	var jsonInfo JsonInfo
	err = json.Unmarshal(b, &jsonInfo)
	if err != nil {
		return err
	}

	for _, item := range jsonInfo.ExtName {
		info.extName[item.Ext] = item.Name
	}
	for _, item := range jsonInfo.NameIcon {
		info.nameIcon[item.Name] = item.Icon
	}
	info.previewLimit = jsonInfo.Preview.Limit
	info.previewList = jsonInfo.Preview.List
	info.address = jsonInfo.Address
	info.passwd = jsonInfo.Passwd
	return nil
}

func (info *Info) Name(ext string) string {
	if a, ok := info.extName[strings.ToLower(ext)]; ok {
		return a
	}
	return "文件"
}

func (info *Info) Icon(name string) string {
	if a, ok := info.nameIcon[name]; ok {
		return a
	}
	return "fa-file-o"
}

func (info *Info) GetNameAndIcon(pathstr string) (string, string) {
	var name string
	if goutil.IsDir(pathstr) {
		name = "目录"
	} else {
		name = info.Name(filepath.Ext(pathstr))
	}
	return name, info.Icon(name)
}

func (info *Info) EnablePreview(ext string, bsize int64) bool {
	if bsize < info.previewLimit {
		ext = strings.ToLower(ext)
		if i := goutil.StringsIndexOf(info.previewList, ext); i >= 0 {
			return true
		}
	}
	return false
}

func (info *Info) PreviewUrl(ext string, bsize int64, url string) string {
	if info.EnablePreview(ext, bsize) {
		return fmt.Sprintf("http://ow365.cn/?i=23123&furl=http://f.ningto.com%s", url)
	}
	return ""
}

func (info *Info) Address() string {
	return info.address
}

func (info *Info) Passwd() string {
	return info.passwd
}
