package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	URL "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mmpx12/optionparser"
)

var (
	success   = 0
	mu        = &sync.Mutex{}
	thread    = make(chan struct{}, 50)
	wg        sync.WaitGroup
	output    = "found_env.txt"
	proxy     string
	insecure  bool
	version   = "alpha:dev"
	userAgent = "Mozilla/5.0 (X11; Linux x86_64)"
	path      = []string{"/.env"}
)

func CheckEnv(client *http.Client, url, path string) {
	defer wg.Done()
	req, err := http.NewRequest("GET", "https://"+url+path, nil)
	if err != nil {
		<-thread
		return
	}
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		<-thread
		return
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	body, _ := ioutil.ReadAll(resp.Body)
	match, _ := regexp.MatchString(`(?m)<body|<script|<html>`, string(body))
	if len(body) >= 2500 || match {
		fmt.Println("\nmatchhhh\n")
		<-thread
		return
	}
	r := regexp.MustCompile(`(?m)^([A-Za-z0-9-]|[-_]|^[\s\.]){1,30}[\s]?=.{1,60}`)
	if r.MatchString(string(body)) {
		all := r.FindAllString(string(body), -1)
		if len(all) > 5 {
			success++
			mu.Lock()
			WriteToFile(resp.Request.URL.String())
			mu.Unlock()
			fmt.Println("\033[1K\rENV FOUND:\033[36m", resp.Request.URL.String(), "\033[0m")
			for _, j := range all {
				v := strings.Split(j, "=")
				fmt.Println("\033[33m" + v[0] + "\033[0m=\033[35m" + v[1])
			}
			<-thread
			return
		}
	}
	<-thread
}

func WriteToFile(target string) {
	f, _ := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	defer f.Close()
	fmt.Fprintln(f, target)
}

func LineNBR(f string) int {
	r, _ := os.Open(f)
	defer r.Close()
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}
	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)
		switch {
		case err == io.EOF:
			return count
		case err != nil:
			return 0
		}
	}
}

func main() {
	var threads, input string
	var printversion bool
	op := optionparser.NewOptionParser()
	op.Banner = "Scan for exposed git repos\n\nUsage:\n"
	op.On("-t", "--thread NBR", "Number of threads (default 50)", &threads)
	op.On("-o", "--output FILE", "Output file (default found_git.txt)", &output)
	op.On("-i", "--input FILE", "Input file", &input)
	op.On("-k", "--insecure", "Ignore certificate errors", &insecure)
	op.On("-u", "--user-agent USR", "Set user agent", &userAgent)
	op.On("-p", "--proxy PROXY", "Use proxy (proto://ip:port)", &proxy)
	op.On("-V", "--version", "Print version and exit", &printversion)
	op.Parse()
	fmt.Printf("\033[31m")
	op.Logo("[X-env]", "doom", false)
	fmt.Printf("\033[0m")

	if printversion {
		fmt.Println("version:", version)
		os.Exit(1)
	}

	if threads != "" {
		tr, _ := strconv.Atoi(threads)
		thread = make(chan struct{}, tr)
	}

	if input == "" {
		fmt.Println("\033[31m[!] You must specify an input file\033[0m\n")
		op.Help()
		os.Exit(1)
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: insecure},
		},
	}
	if proxy != "" {
		proxyURL, _ := URL.Parse(proxy)
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	log.SetOutput(io.Discard)
	os.Setenv("GODEBUG", "http2client=0")
	readFile, err := os.Open(input)
	defer readFile.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	i := 0
	total := LineNBR(input)
	for fileScanner.Scan() {
		i++
		target := fileScanner.Text()
		fmt.Printf("\033[1K\r\033[31m[\033[33m%d\033[36m/\033[33m%d \033[36m(\033[32m%d\033[36m)\033[31m] \033[35m%s\033[0m", i, total, success, target)
		for _, p := range path {
			thread <- struct{}{}
			wg.Add(1)
			go CheckEnv(client, target, p)
		}
	}
	wg.Wait()
	fmt.Printf("\033[1K\rFound %d git repos.\n", success)
}
