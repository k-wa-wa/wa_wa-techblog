package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/joho/godotenv"
)

var (
	MICROCMS_BASE_URL  string
	X_MICROCMS_API_KEY string
	MD_OUTPUT_DIR      string
)

func init() {
	godotenv.Load("./.env")
	MICROCMS_BASE_URL = os.Getenv("MICROCMS_BASE_URL")
	X_MICROCMS_API_KEY = os.Getenv("X_MICROCMS_API_KEY")
	MD_OUTPUT_DIR = os.Getenv("MD_OUTPUT_DIR")
}

func main() {

	if err := PullMicrocmsBlogs(); err != nil {
		log.Fatal(err)
	}
}

type MicrocmsBlog struct {
	Id          string `json:id`
	CreatedAt   string `json:createdAt`
	UpdatedAt   string `json:updatedAt`
	PublishedAt string `json:publishedAt`
	RevisedAt   string `json:revisedAt`
	Title       string `json:title`
	Body        string `json:body`
	Tags        []struct {
		TagName string `json:tagName`
	} `json:tags`
	Image struct {
		Url    string `json:url`
		Height int    `json:height`
		Width  int    `json:width`
	} `json:image`
}

type MicrocmsBlogRes struct {
	Contents []MicrocmsBlog `json:contents`
}

func getContents(offset int, limit int) ([]MicrocmsBlog, error) {
	url := fmt.Sprintf("%s?offset=%d&limit=%d", MICROCMS_BASE_URL, offset, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MICROCMS-API-KEY", X_MICROCMS_API_KEY)

	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf(strconv.Itoa(res.StatusCode))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var microcmsBlogRes MicrocmsBlogRes
	if err := json.Unmarshal(body, &microcmsBlogRes); err != nil {
		return nil, err
	}
	return microcmsBlogRes.Contents, nil
}

func htmlToMarkdown(html string) (error, string) {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return err, ""
	}
	return nil, markdown
}

func (blog *MicrocmsBlog) toHugoRobustMarkdown() (error, string) {
	err, markdownBody := htmlToMarkdown(blog.Body)
	if err != nil {
		return err, ""
	}
	return nil, fmt.Sprintf(`
+++
date = "%s"
title = "%s"
thumbnail = "%s"
+++

%s
	`, blog.CreatedAt, blog.Title, blog.Image.Url, markdownBody)
}

func (blog *MicrocmsBlog) toMarkdown(filename string) error {
	err, markdown := blog.toHugoRobustMarkdown()
	file, err := os.Create(filepath.Join(MD_OUTPUT_DIR, blog.Id+".md"))
	if err != nil {
		return err
	}
	if _, err := file.Write([]byte(markdown)); err != nil {
		return err
	}
	return nil
}

func PullMicrocmsBlogs() error {
	os.MkdirAll(MD_OUTPUT_DIR, os.ModePerm)

	LIMIT := 10
	offset := 0
	for {
		blogs, err := getContents(offset, LIMIT)
		if err != nil {
			return err
		}
		if len(blogs) == 0 {
			break
		}
		for _, blog := range blogs {
			filename := blog.Id + ".md"
			if err := blog.toMarkdown(filename); err != nil {
				log.Print(err)
				continue
			}
		}
		offset += LIMIT
	}
	return nil
}
