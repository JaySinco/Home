package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {
	devices := map[string]string{
		"lenovopc-jaysinco":   "Zhou Xinke",
		"Honor_8":             "Zhou Xinke",
		"androidpad-jaysinco": "Zhou Xinke",
		"AriannadeiPhone":     "Dong Shen",
		"Wenqide-iPhone":      "Gou Wenqi",
		"taiyuede-iPad":       "Zhong Qijia",
		"ZQJdeMBP":            "Zhong Qijia",
		"ZQJ-iPhone7":         "Zhong Qijia",
	}
	peers := make(map[string]bool)
	logged := make(map[string]bool)
	stok := getStok()
	go func() {
		bufio.NewScanner(os.Stdin).Scan()
		os.Exit(0)
	}()
	for {
		for _, dv := range whoInHome(stok) {
			pr, ok := devices[dv]
			if !ok {
				if !logged[dv] {
					fmt.Printf("\runknow device: %s\n", dv)
					logged[dv] = true
				}
			} else {
				peers[pr] = true
			}
		}
		fmt.Print("\r")
		for pr, _ := range peers {
			fmt.Printf("%s; ", pr)
		}
		time.Sleep(1 * time.Second)
	}
}

func getStok() string {
	resp, _ := http.Post(
		"http://192.168.0.1/",
		"application/json; charset=UTF-8",
		strings.NewReader(`{"method":"do","login":{"password":"7Bq7xghc9TefbwK"}}`),
	)
	body, _ := ioutil.ReadAll(resp.Body)
	reg := regexp.MustCompile(`"stok":"(.*?)"`)
	stok := reg.FindStringSubmatch(string(body))[1]
	return stok
}

func whoInHome(stok string) []string {
	resp, _ := http.Post(
		fmt.Sprintf("http://192.168.0.1/stok=%s/ds", stok),
		"application/json; charset=UTF-8",
		strings.NewReader(`{"hosts_info":{"table":"online_host"},"network":{"name":"iface_mac"},"method":"get"}`),
	)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	reg := regexp.MustCompile(`"hostname": "(.*?)"`)
	hosts := reg.FindAllStringSubmatch(string(body), -1)
	peer := make([]string, 0)
	for _, host := range hosts {
		hostName, _ := url.QueryUnescape(host[1])
		peer = append(peer, hostName)
	}
	return peer
}
