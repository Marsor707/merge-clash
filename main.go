package main

import (
	_ "embed"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

//go:embed default.yml
var template []byte
var clash *Clash

type Clash struct {
	Port               int              `yaml:"port"`
	SocksPort          int              `yaml:"socks-port"`
	RedirPort          int              `yaml:"redir-port"`
	Mode               string           `yaml:"mode"`
	ExternalController string           `yaml:"external-controller"`
	AllowLan           bool             `yaml:"allow-lan"`
	Proxies            []map[string]any `yaml:"proxies"`
	ProxyGroups        []ProxyGroup     `yaml:"proxy-groups"`
	RuleProviders      map[string]any   `yaml:"rule-providers"`
	Rules              []string         `yaml:"rules"`
}

type ProxyGroup struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
}

type SourceFiles []string

func (s *SourceFiles) String() string {
	return fmt.Sprintf("%+v", *s)
}

func (s *SourceFiles) Set(file string) error {
	*s = append(*s, file)
	return nil
}

var (
	files SourceFiles
	port  int
	out   string
)

func init() {
	_ = yaml.Unmarshal(template, &clash)
	clash.Proxies = make([]map[string]any, 0)
	clash.ProxyGroups = make([]ProxyGroup, 1)
	clash.ProxyGroups[0] = ProxyGroup{
		Name:    "PROXY",
		Type:    "select",
		Proxies: make([]string, 0),
	}
	flag.Var(&files, "f", "源clash文件")
	flag.IntVar(&port, "p", 0, "当http服务使用时端口")
	flag.StringVar(&out, "o", "template.yml", "当输出文件时目标文件名")
	flag.Parse()
}

func main() {
	for _, file := range files {
		if err := AddToClash(file); err != nil {
			fmt.Println(err)
			return
		}
	}
	if port != 0 {
		ServeHttp(port)
	} else {
		if err := WriteClash(out); err != nil {
			fmt.Println(err)
		}
	}
}

func parseProxyFile(file string) ([]map[string]any, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	type pro struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	var p pro
	err = yaml.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p.Proxies, nil
}

func AddToClash(file string) error {
	proxies, err := parseProxyFile(file)
	if err != nil {
		return err
	}
	for i := 0; i < len(proxies); i++ {
		name := formatName(proxies[i]["name"].(string))
		proxies[i]["name"] = name
		clash.Proxies = append(clash.Proxies, proxies[i])
		clash.ProxyGroups[0].Proxies = append(clash.ProxyGroups[0].Proxies, name)
	}
	return nil
}

func formatName(source string) string {
	return strings.Trim(source, " ")
}

func WriteClash(name string) error {
	data, err := yaml.Marshal(clash)
	if err != nil {
		return err
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func ServeHttp(port int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := yaml.Marshal(clash)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=template.yml")
		w.WriteHeader(200)
		_, _ = w.Write(data)
	})
	_ = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
