package main

import (
	"clash/converter"
	_ "embed"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"net"
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

type StringSet []string

func (s *StringSet) String() string {
	return fmt.Sprintf("%+v", *s)
}

func (s *StringSet) Set(file string) error {
	*s = append(*s, file)
	return nil
}

var (
	files        StringSet
	urls         StringSet
	port         int
	out          string
	subConverter string
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
	flag.Var(&urls, "u", "订阅地址")
	flag.IntVar(&port, "p", 0, "当http服务使用时端口")
	flag.StringVar(&out, "o", "template.yml", "当输出文件时目标文件名")
	flag.StringVar(&subConverter, "s", "", "subConverter后端地址")
	flag.Parse()
}

func main() {
	for _, file := range files {
		if err := AddWithFile(file); err != nil {
			fmt.Println(err)
			return
		}
	}
	if len(urls) > 0 {
		if subConverter != "" {
			if err := AddWithSubConverter(urls); err != nil {
				fmt.Println(err)
				return
			}
		} else {
			if err := AddWithInnerConverter(urls); err != nil {
				fmt.Println(err)
				return
			}
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

func AddWithFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	return addToClash(data)
}

func AddWithSubConverter(urls []string) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/sub", subConverter), nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Set("target", "clash")
	q.Set("url", strings.Join(urls, "|"))
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return addToClash(data)
}

func AddWithInnerConverter(urls []string) error {
	for i := 0; i < len(urls); i++ {
		proxies, err := converter.ParseSubscribe(urls[i])
		if err != nil {
			return err
		}
		addProxiesToClash(proxies)
	}
	return nil
}

func addToClash(data []byte) error {
	type pro struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	var p pro
	err := yaml.Unmarshal(data, &p)
	if err != nil {
		return err
	}
	addProxiesToClash(p.Proxies)
	return nil
}

func addProxiesToClash(proxies []map[string]any) {
	for i := 0; i < len(proxies); i++ {
		name := formatName(proxies[i]["name"].(string))
		proxies[i]["name"] = name
		clash.Proxies = append(clash.Proxies, proxies[i])
		clash.ProxyGroups[0].Proxies = append(clash.ProxyGroups[0].Proxies, name)
	}
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
	fmt.Printf("访问 http://%s:%d 进行订阅\n", getLocalIp(), port)
	_ = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func getLocalIp() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "localhost"
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return strings.Split(localAddr.String(), ":")[0]
}
