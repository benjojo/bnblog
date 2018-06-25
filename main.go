package bnblog

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/codegangsta/martini"
	"github.com/russross/blackfriday"

	"appengine"
	"appengine/datastore"
)

var PostTemplate = template.Must(template.ParseFiles("public2/pagetempl.html"))
var HomeTemplate = template.Must(template.ParseFiles("public2/hometempl.html"))

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
	m.Get("/raw/:name", ReadRawPost)
	m.Post("/admin/new", PublishPost)
	m.Get("/admin/", Admin)
	m.Get("/", ListPosts)
	m.Get("/all", ListPosts)
	m.Get("/rss.xml", GetRSS)
	m.Get("/sitemap.xml", GetSitemap)
	m.Get("/admin/run_gc", Run_GC)
	m.Get("/admin/backup.tar", Producebackup)
	m.Get("/admin/remove/:name", RemovePost)
	m.Post("/admin/uploadfile", UploadFile)

	m.Get("/lessons/:year/:month/:day/:title", MigrateOldURLS)
	m.Get("/lessons/:year/:month/:day/:title/", MigrateOldURLS)
	m.Get("/errors/:year/:month/:day/:title", MigrateOldURLS)
	m.Get("/errors/:year/:month/:day/:title/", MigrateOldURLS)
	m.Get("/posts/errors/:year/:month/:day/:title", MigrateOldURLS)
	m.Get("/posts/errors/:year/:month/:day/:title/", MigrateOldURLS)
	m.Get("/asset/:tag", ReadFile)

	m.Use(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Cache-Control", "public")
		res.Header().Add("X-Served-By", GimmeDC(res, req))
		res.Header().Add("X-Served-For", req.Header.Get("CF-RAY"))
	})

	m.Use(martini.Static("public2"))

	http.Handle("/", m)
}

func MigrateOldURLS(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	http.Redirect(rw, req, fmt.Sprintf("https://blog.benjojo.co.uk/post/%s-%s-%s-%s.md", params["year"], params["month"], params["day"], params["title"]), http.StatusMovedPermanently)
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
		if fmt.Sprint(err) == "datastore: no such entity" {
			http.Error(rw, "This blog post cannot be found, Please check your URL", http.StatusNotFound)
			return
		}

		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	postd, _ := base64.StdEncoding.DecodeString(post.Content)
	// post.Content = strings.Replace(string(postd), "\n", "\r\n\r\n", -1)
	post.Content = string(postd)
	output := blackfriday.MarkdownCommon([]byte(post.Content))
	lines := strings.Split(string(postd), "\n")
	layoutData := struct {
		Title   string
		Content string
		Date    string
	}{
		Title:   lines[0],
		Content: string(output),
		Date:    post.Date.Format("Jan 2 2006"),
	}

	err = PostTemplate.Execute(rw, layoutData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func ReadRawPost(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	c := appengine.NewContext(req)
	k := datastore.NewKey(c, "Post", params["name"], 0, nil)
	post := Post{}
	err := datastore.Get(c, k, &post)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	postd, _ := base64.StdEncoding.DecodeString(post.Content)
	rw.Write(postd)
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
