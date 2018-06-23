package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var dataDir = flag.String("d", "D:/Jaysinco/Python/data/chinese-poetry/json/", "data dir path")
var filePat = flag.String("f", ".", "file name regexp pattern")
var maxNum = flag.Int("n", 1e9, "poem number limit")

func main() {
	flag.Parse()
	set := make([]string, 0)
	total := 0
	sep := "::"
	opencc := newOpencConv("./opencc/config/t2s.json")
	defer opencc.close()

	fl, err := getFileList()
	if err != nil {
		fmt.Printf("get file list: %v\n", err)
		return
	}
	fmt.Printf("Sourcing %d files, ", len(fl))

ReadLoop:
	for _, filename := range fl {
		fp, err := os.Open(filename)
		if err != nil {
			fmt.Printf("open file: %v\n", err)
			return
		}
		defer fp.Close()
		decoder := json.NewDecoder(fp)
		pm := make([]*poetry, 0)
		if err := decoder.Decode(&pm); err != nil {
			fmt.Printf("decode json: %v\n", err)
			return
		}

		for _, p := range pm {
			if p.Title == "" || strings.Contains(p.Title, " ") || p.Author == "" ||
				len(p.Paragraphs) == 0 || len(p.Paragraphs[0]) == 0 {
				continue
			}
			var poem bytes.Buffer
			poem.WriteString(p.Title)
			poem.WriteString(sep)
			poem.WriteString(p.Author)
			poem.WriteString(sep)
			for _, n := range p.Paragraphs {
				poem.WriteString(n)
			}
			cc := poem.String()
			if strings.ContainsAny(cc, "（）《》□[]") {
				continue
			}
			set = append(set, opencc.convert(cc))
			total++
			if total >= *maxNum {
				break ReadLoop
			}
		}
	}

	fd, err := os.Create("poem.txt")
	if err != nil {
		fmt.Printf("create file: %v\n", err)
		return
	}
	defer fd.Close()
	fd.WriteString(strings.Join(set, "\n"))
	fmt.Printf("total %d poems readed.\n", total)
}

func getFileList() ([]string, error) {
	fs, err := ioutil.ReadDir(*dataDir)
	if err != nil {
		return nil, err
	}
	fl := make([]string, 0)
	for _, fi := range fs {
		if ok, err := regexp.MatchString(*filePat, fi.Name()); err == nil && ok && !fi.IsDir() {
			fl = append(fl, *dataDir+fi.Name())
		}
	}
	return fl, nil
}

type poetry struct {
	Title      string
	Author     string
	Paragraphs []string
}
