package converter

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

type Outbound struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Server  string `yaml:"server"`
	Port    int    `yaml:"port"`
	Uuid    string `yaml:"uuid"`
	AlterId int    `yaml:"alterId"`
	Cipher  string `yaml:"cipher"`
	Network string `yaml:"network"`
	Udp     bool   `yaml:"udp"`
}

func (o Outbound) ToMap() map[string]any {
	o.Udp = true
	result := make(map[string]any)
	objValue := reflect.ValueOf(o)
	objType := objValue.Type()
	for i := 0; i < objValue.NumField(); i++ {
		field := objValue.Field(i)
		fieldName := objType.Field(i).Tag.Get("yaml")
		result[fieldName] = field.Interface()
	}
	return result
}

type Vmess struct {
	V    string `json:"v"`
	Ps   string `json:"ps"`
	Add  string `json:"add"`
	Port string `json:"port"`
	Id   string `json:"id"`
	Aid  string `json:"aid"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	Path string `json:"path"`
	Tls  string `json:"tls"`
}

func (v Vmess) ToOutbound() *Outbound {
	return &Outbound{
		Name:    v.Ps,
		Type:    "vmess",
		Server:  v.Add,
		Port:    parseInt(v.Port),
		Uuid:    v.Id,
		AlterId: parseInt(v.Aid),
		Cipher:  "auto",
		Network: v.Net,
	}
}

func parseInt(str string) int {
	v, _ := strconv.Atoi(str)
	return v
}

func ParseSubscribe(link string) ([]map[string]any, error) {
	data, err := doRequest(link)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	res := make([]map[string]any, 0)
	for scanner.Scan() {
		proto, err := url.Parse(scanner.Text())
		if err != nil {
			return nil, err
		}
		if proto.Scheme == "vmess" {
			outbound, err := parseVmess(proto.Host)
			if err != nil {
				return nil, err
			}
			res = append(res, outbound.ToMap())
		} else if proto.Scheme == "ss" {
			// todo
		} else if proto.Scheme == "trojan" {
			// todo
		}
	}
	return res, nil
}

func parseVmess(data string) (*Outbound, error) {
	vm, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	var vms Vmess
	if err = json.Unmarshal(vm, &vms); err != nil {
		return nil, err
	}
	return vms.ToOutbound(), nil
}

func doRequest(link string) ([]byte, error) {
	fmt.Println("开始下载订阅", link)
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println("订阅下载完成")
	return data, nil
}
