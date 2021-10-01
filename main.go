package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

type form struct {
	action  string
	method  string
	enctype string
	inputs  []string
}

var (
	concurrency int
	timeout int
	findedForm = make([]string, 0)
)

func main() {
	flag.IntVar(&concurrency, "c", 50, "Set the Concurrency")
	flag.IntVar(&timeout, "timeout", 60, "Set timeout")
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				link := scanner.Text()
				find_forms(link)
			}
			wg.Done()
		}()
		wg.Wait()
	}
}

func find_reflected(form_data form) {
	for _, input := range form_data.inputs {
		payload := "FORMECCEPT"
		_, body, raw := makeRequest(form_data, payload, input)
		re := regexp.MustCompile(payload)
		match := re.FindStringSubmatch(body)
		if match != nil {
			fileName := "/tmp/ftf"+strconv.Itoa(rand.Int())
			file, _ := os.Create(fileName)
			file.Write(raw)
			fmt.Println(fileName)
		}
	}
} 

func makeRequest(form_data form, payload string, param string) (requestDp *http.Response, bodyString string, raw_request []byte){
	data := url.Values{}

	var multipartData bytes.Buffer
	w := multipart.NewWriter(&multipartData)
	for _, value := range form_data.inputs {
		if (value == param) {
			w.WriteField(value, payload)
			data.Set(value, payload)
		} else {
			data.Set(value, "test")
			w.WriteField(value, "test")
		}
	}
	w.Close()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	} 
	req, _ := http.NewRequest(strings.ToUpper(form_data.method), form_data.action, strings.NewReader(data.Encode()))
	if (form_data.enctype == "multipart/form-data") {
		req, _ = http.NewRequest(strings.ToUpper(form_data.method), form_data.action, &multipartData)
	}
	
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36")
	req.Header.Add("Content-type", form_data.enctype)

	requestDump, _ := httputil.DumpRequest(req, true)

	resp, _ := client.Do(req)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	
	return resp, string(bodyBytes), requestDump
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func find_forms(url string) {
	c := colly.NewCollector()
	c.OnHTML("form", func(e *colly.HTMLElement) {
		action := e.Attr("action")
		method := e.Attr("method")
		enctype := e.Attr("enctype")

		if method == "" {
			method = "GET"
		}

		if action == "" {
			action = url
		}

		if enctype == "" {
			enctype = "application/x-www-form-urlencoded"
		}

		var inputStrings []string
		e.ForEach("input", func(_ int, el *colly.HTMLElement) {
			if (el.Attr("name") != "") {
				inputStrings = append(inputStrings, el.Attr("name"))
			}
		})

		absolute := e.Request.AbsoluteURL(action)

		l := true
		for _, v := range findedForm {
			if (v == absolute) {
				l = false
			}
		}
		if (l) {
			findedForm = append(findedForm, absolute)
			frm := form{action: absolute, enctype: enctype, method: method, inputs: inputStrings}
			find_reflected(frm)
		}
		
	})

	c.Visit(url)
}
