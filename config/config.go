package config

import (
	"encoding/json"
	"gonetdisk/util"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
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
}

type Info struct {
	extName  map[string]string
	nameIcon map[string]string
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
	if util.IsDir(pathstr) {
		name = "目录"
	} else {
		name = info.Name(filepath.Ext(pathstr))
	}
	return name, info.Icon(name)
}
