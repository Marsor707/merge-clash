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
	"strings"
)

type Outbound struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Server   string `yaml:"server"`
	Port     int    `yaml:"port"`
	Uuid     string `yaml:"uuid"`
	AlterId  *int   `yaml:"alterId"`
	Cipher   string `yaml:"cipher"`
	Network  string `yaml:"network"`
	Password string `yaml:"password"`
	Sni      string `yaml:"sni"`
	Udp      bool   `yaml:"udp"`
}

func (o Outbound) ToMap() map[string]any {
	o.Udp = true
	result := make(map[string]any)
	objValue := reflect.ValueOf(o)
	objType := objValue.Type()
	for i := 0; i < objValue.NumField(); i++ {
		field := objValue.Field(i)
		if field.IsZero() {
			continue
		}
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
	var aid = parseInt(v.Aid)
	return &Outbound{
		Name:    v.Ps,
		Type:    "vmess",
		Server:  v.Add,
		Port:    parseInt(v.Port),
		Uuid:    v.Id,
		AlterId: &aid,
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
		var outbound *Outbound
		if proto.Scheme == "vmess" {
			outbound, err = parseVmess(proto.Host)
			if err != nil {
				return nil, err
			}
		} else if proto.Scheme == "ss" {
			outbound, err = parseSS(proto)
			if err != nil {
				return nil, err
			}
		} else if proto.Scheme == "trojan" {
			outbound, err = parseTrojan(proto)
			if err != nil {
				return nil, err
			}
		}
		res = append(res, outbound.ToMap())
	}
	return res, nil
}

func parseTrojan(proto *url.URL) (*Outbound, error) {
	host := strings.Split(proto.Host, ":")
	outbound := &Outbound{
		Name:     proto.Fragment,
		Type:     "trojan",
		Server:   host[0],
		Port:     parseInt(host[1]),
		Password: proto.User.Username(),
		Sni:      proto.Query().Get("sni"),
	}
	return outbound, nil
}

func parseSS(proto *url.URL) (*Outbound, error) {
	secret, err := base64.StdEncoding.DecodeString(proto.User.Username())
	if err != nil {
		return nil, err
	}
	secretInfo := strings.Split(string(secret), ":")
	host := strings.Split(proto.Host, ":")
	outbound := &Outbound{
		Name:     proto.Fragment,
		Type:     "ss",
		Server:   host[0],
		Port:     parseInt(host[1]),
		Cipher:   secretInfo[0],
		Password: secretInfo[1],
	}
	return outbound, nil
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
