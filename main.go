// Downloads all the artifacts from a given Jenkins build
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

type api struct {
	Building  bool
	Result    string
	Artifacts []struct {
		FileName     string
		RelativePath string
	}
	URL string
}

func main() {
	expStr := flag.String("match", "", "Regexp to match file names against")
	flag.Parse()
	url := flag.Arg(0)

	var exp *regexp.Regexp
	if *expStr != "" {
		exp = regexp.MustCompile(*expStr)
	}

	resp, err := http.Get(url + "/api/json")
	if err != nil {
		log.Fatal(err)
	}

	var res api
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Fatal(err)
	}

	log.Println("Build result is", res.Result)
	if res.Result != "SUCCESS" {
		os.Exit(1)
	}

	var wg sync.WaitGroup
	for _, art := range res.Artifacts {
		file := art.FileName
		relPath := art.RelativePath
		if exp != nil && !exp.MatchString(file) {
			continue
		}
		wg.Add(1)
		go func() {
			if err := download(res.URL, relPath); err != nil {
				log.Println("Download of", file, "failed:", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func download(url, file string) error {
	short := filepath.Base(file)
	log.Println("Downloading", short)

	path := url + "/artifact/" + file
	resp, err := http.Get(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	fd, err := os.Create(short)
	if err != nil {
		return err
	}

	if _, err := io.Copy(fd, resp.Body); err != nil {
		fd.Close()
		os.Remove(file)
		return err
	}

	return fd.Close()
}
