package bnblog

import (
	"appengine"
	"appengine/datastore"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/feeds"
	"net/http"
	"strings"
	"time"
)

func GetRSS(rw http.ResponseWriter, req *http.Request) {
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "benjojo blog",
		Link:        &feeds.Link{Href: "https://blog.benjojo.co.uk"},
		Description: "Programming, Networking and some things I found hard to fix at some point",
		Author:      &feeds.Author{"Ben Cartwright-Cox", "ben@benjojo.co.uk"},
		Created:     now,
	}

	c := appengine.NewContext(req)
	q := datastore.NewQuery("Post").Order("-Date").Limit(100)
	posts := make([]Post, 0, 100)

	if _, err := q.GetAll(c, &posts); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	feed.Items = []*feeds.Item{}

	for _, v := range posts {
		if !strings.HasPrefix(v.Slug, "DRAFT-") {
			// newpost := PostFormatted{
			// 	Author:  v.Author,
			// 	Content: v.Content,
			// 	Date:    v.Date.Format("2006-01-02 15:04:05"),
			// 	Slug:    v.Slug,
			// 	Title:   v.Title,
			// }
			// FormattedPosts = append(FormattedPosts, newpost)
			postd, _ := base64.StdEncoding.DecodeString(v.Content)
			wot := &feeds.Item{
				Title:       v.Title,
				Link:        &feeds.Link{Href: fmt.Sprintf("https://blog.benjojo.co.uk/post/%s", v.Slug)},
				Description: string(postd[:256]),
				Author:      &feeds.Author{"Ben Cox", "ben@benjojo.co.uk"},
				Created:     v.Date,
			}
			feed.Items = append(feed.Items, wot)
		}
	}

	rss, err := feed.ToRss()
	if err != nil {
		http.Error(rw, fmt.Sprintf("argh %s", err), http.StatusInternalServerError)
		return
	}
	rw.Header().Add("Content-Type", "application/rss+xml")
	rw.Write([]byte(rss))
}
