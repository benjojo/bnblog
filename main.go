package bnblog

import (
	"appengine"
	"appengine/datastore"
	"encoding/base64"
	"github.com/codegangsta/martini"
	"github.com/russross/blackfriday"
	"net/http"
	"strings"
	"text/template"
	"time"
)

var PostTemplate = template.Must(template.ParseFiles("public/pagetempl.html"))
var HomeTemplate = template.Must(template.ParseFiles("public/hometempl.html"))

type Post struct {
	Author  string
	Content string `datastore:",noindex"`
	Date    time.Time
	Slug    string
	Title   string
}

type PostFormatted struct {
	Author  string
	Content string `datastore:",noindex"`
	Date    string
	Slug    string
	Title   string
}

func init() {
	m := martini.Classic()
	m.Get("/post/:name", ReadPost)
	m.Post("/admin/new", PublishPost)
	m.Get("/admin/", Admin)
	m.Get("/", ListPosts)
	m.Get("/all", ListPosts)
	http.Handle("/", m)
}

func ReadPost(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	// c := appengine.NewContext(r)
	// key := datastore.NewIncompleteKey(c, "Greeting", PostKey(c))
	// _, err := datastore.Put(c, key, &g)

	c := appengine.NewContext(req)
	k := datastore.NewKey(c, "Post", params["name"], 0, nil)
	post := Post{}
	err := datastore.Get(c, k, &post)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	postd, _ := base64.StdEncoding.DecodeString(post.Content)
	// post.Content = strings.Replace(string(postd), "\n", "\r\n\r\n", -1)
	post.Content = string(postd)
	output := blackfriday.MarkdownBasic([]byte(post.Content))
	lines := strings.Split(string(postd), "\n")
	layoutData := struct {
		Title   string
		Content string
	}{
		Title:   lines[0],
		Content: string(output),
	}

	err = PostTemplate.Execute(rw, layoutData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func ListPosts(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	q := datastore.NewQuery("Post").Order("-Date").Limit(100)
	posts := make([]Post, 0, 100)

	if _, err := q.GetAll(c, &posts); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	FormattedPosts := make([]PostFormatted, 0)

	for _, v := range posts {
		if !strings.HasPrefix(v.Slug, "DRAFT-") {
			newpost := PostFormatted{
				Author:  v.Author,
				Content: v.Content,
				Date:    v.Date.Format("2006-01-02 15:04:05"),
				Slug:    v.Slug,
				Title:   v.Title,
			}
			FormattedPosts = append(FormattedPosts, newpost)
		}
	}

	layoutData := struct {
		Posts []PostFormatted
	}{
		Posts: FormattedPosts,
	}

	err := HomeTemplate.Execute(rw, layoutData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

}
