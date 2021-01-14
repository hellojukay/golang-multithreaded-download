package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"
)

var u string
var thread int
var filename string

func init() {
	flag.StringVar(&u, "url", "", "http[s] link url")
	flag.StringVar(&filename, "o", "", "filename ")
	flag.Parse()
}

type Range struct {
	Start int64
	End   int64
}

type Downloader struct {
	Url  string
	File string
	fh   *os.File
	wg   sync.WaitGroup
}

// 下载文件，等待所有 goroutine 返回
func (loader *Downloader) Down(thread int) error {
	// 链接服务器，获取文件大小
	res, err := http.Get(loader.Url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if filename == "" {
		filename = path.Base(res.Request.URL.Path)
		_, params, err := mime.ParseMediaType(res.Header.Get("Content-Disposition"))

		if err == nil {
			if params["filename"] != "" {
				filename = params["filename"]
			}
		}
	}
	log.Printf("download file from %s,content length %d\n", loader.Url, res.ContentLength)
	fh, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		if err != nil {
			return err
		}
	}
	log.Printf("downloading file %s\n", filename)
	defer fh.Close()
	defer fh.Sync()
	loader.fh = fh
	ranges, err := loader.getRange(res.ContentLength)
	if err != nil {
		return err
	}
	for _, r := range ranges {
		// 为每一个 goroutine 分配任务
		func() {
			defer loader.wg.Add(1)
			go loader.down(r)
		}()
	}
	loader.wg.Wait()
	return nil
}

// 下载分片
func (loader *Downloader) down(r Range) error {
	defer loader.wg.Add(-1)
	req, err := http.NewRequest("GET", loader.Url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", r.Start, r.End))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("request %s error %s\n", loader.Url, err)
		return err
	}
	defer res.Body.Close()
	var buffer []byte = make([]byte, 1024*1024)
	var off = r.Start
	for {
		n, err := res.Body.Read(buffer)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = loader.fh.WriteAt(buffer[:n], off)
		if err != nil {
			log.Printf("write to file error %s\n", err)
			os.Exit(1)
		}
		off = off + int64(n)
	}
	return nil
}

// 获取分配下载区间
func (loader *Downloader) getRange(contentLength int64) ([]Range, error) {
	var result []Range
	avg := contentLength / int64(runtime.NumCPU())
	var start, end int64
	for {
		if int64(start+avg) >= contentLength {
			end = contentLength
			result = append(result, Range{Start: start, End: end})
			break
		} else {
			end = start + avg
			result = append(result, Range{Start: start, End: end})
		}
		start = end + 1
	}
	return result, nil
}

func main() {
	log.Printf("start downloading...\n")
	var loader = &Downloader{
		Url:  u,
		File: "",
	}
	err := loader.Down(thread)
	if err != nil {
		log.Printf("download from %s failed,%s\n", u, err)
		os.Exit(1)
	}
}
