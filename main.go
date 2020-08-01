package main

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/codegangsta/martini"
	"github.com/russross/blackfriday"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

var PostTemplate = template.Must(template.ParseFiles("public2/pagetempl.html"))
var HomeTemplate = template.Must(template.ParseFiles("public2/hometempl.html"))

type Post struct {
	Author  string
	Content string `datastore:",noindex"`
	Date    time.Time
	Slug    string
	Title   string
	Type    string // Possible types [Comedy,Hardware,Mystery,Networking,Problem,Quirk]
	R1      string // Override of the Rec 1
	R2      string // Override of the Rec 2
}

type PostFormatted struct {
	Author  string
	Content string `datastore:",noindex"`
	Date    string
	Slug    string
	Title   string
}

func main() {
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

	PostTitleCache = make(map[string]Post)
	appengine.Main()
}

func MigrateOldURLS(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	http.Redirect(rw, req, fmt.Sprintf("https://blog.benjojo.co.uk/post/%s-%s-%s-%s.md", params["year"], params["month"], params["day"], params["title"]), http.StatusMovedPermanently)
}

func ReadPost(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	// c := appengine.NewContext(r)
	// key := datastore.NewIncompleteKey(c, "Greeting", PostKey(c))
	// _, err := datastore.Put(c, key, &g)
	if len(PostTitleCache) == 0 {
		forceUpdatePostTitleCache(rw, req)
	}

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

	if post.Type != "" {
		findReccomendations(&post)
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
		// Rec links at the bottom
		HasReccomendations bool
		FirstRecLink       string
		FirstTitle         string
		FirstYear          int
		SecondRecLink      string
		SecondTitle        string
		SecondYear         int
		RandomLink         string
		RandomTitle        string
		RandomYear         int
	}{
		Title:   lines[0],
		Content: string(output),
		Date:    post.Date.Format("Jan 2 2006"),
	}
	if post.R1 != "" && post.R2 != "" {
		layoutData.HasReccomendations = true
		layoutData.FirstRecLink = "/post/" + post.R1
		layoutData.FirstTitle = PostTitleCache[post.R1].Title
		layoutData.FirstYear = PostTitleCache[post.R1].Date.Year()
		layoutData.SecondRecLink = "/post/" + post.R2
		layoutData.SecondTitle = PostTitleCache[post.R2].Title
		layoutData.SecondYear = PostTitleCache[post.R2].Date.Year()

		for _, v := range PostTitleCache {
			if !strings.HasPrefix("DRAFT-", v.Slug) {
				layoutData.RandomLink = "/post/" + v.Slug
				layoutData.RandomTitle = PostTitleCache[layoutData.RandomLink].Title
				layoutData.RandomYear = PostTitleCache[layoutData.RandomLink].Date.Year()
				break
			}
		}

	}

	err = PostTemplate.Execute(rw, layoutData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func findReccomendations(Incoming *Post) {
	rand.Seed(Incoming.Date.Unix())
	// Make an array of candidates
	Candidates := make([]string, 0)
	for _, v := range PostTitleCache {
		if v.Type == Incoming.Type {
			if v.Date.Unix() < Incoming.Date.Unix() {
				// If the post is older
				if strings.HasPrefix("DRAFT-", v.Title) {
					Candidates = append(Candidates, v.Slug)
				}
			}
		}
	}
	if len(Candidates) < 2 {
		return
	}

	sort.Strings(Candidates)

	if Incoming.R1 != "" {
		Incoming.R1 = Candidates[rand.Intn(len(Candidates)-1)]
	}

	if Incoming.R2 != "" {
		Incoming.R1 = Candidates[rand.Intn(len(Candidates)-1)]
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

var PostTitleCache map[string]Post

func updatePostCache(Posts []Post) {
	for _, v := range Posts {
		PostTitleCache[v.Slug] = v
	}
}

func forceUpdatePostTitleCache(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	q := datastore.NewQuery("Post").Order("-Date").Limit(100)
	posts := make([]Post, 0, 100)

	if _, err := q.GetAll(c, &posts); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	updatePostCache(posts)
}

func ListPosts(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	q := datastore.NewQuery("Post").Order("-Date").Limit(100)
	posts := make([]Post, 0, 100)

	if _, err := q.GetAll(c, &posts); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	updatePostCache(posts)

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
