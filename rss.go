package bnblog

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/feeds"

	"appengine"
	"appengine/datastore"
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
				Author:      &feeds.Author{"ben@benjojo.co.uk", "ben@benjojo.co.uk"},
				Created:     v.Date,
				Id:          generateBadUUID(v.Title),
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

// This is a hack as you might have guessed. This blogging system was
// never designed with UUIDs in mind, so I'm sort of just generating one
// out of the title, MD5 is fine since I don't think I am going attack
// myself with colliding titles.
func generateBadUUID(title string) string {
	hashbytes := md5.Sum([]byte(title))
	return fmt.Sprintf("%1x%1x%1x%1x-%1x%1x-40%1x-%1x%1x-%1x%1x%1x%1x%1x%1x",
		hashbytes[0], hashbytes[1], hashbytes[2], hashbytes[3], hashbytes[4],
		hashbytes[5], hashbytes[6], hashbytes[7], hashbytes[8], hashbytes[9],
		hashbytes[10], hashbytes[11], hashbytes[12], hashbytes[13],
		hashbytes[14])

}
