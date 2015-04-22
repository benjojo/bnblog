package bnblog

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"net/http"
	"strings"
)

func GetSitemap(rw http.ResponseWriter, req *http.Request) {

	rw.Header().Add("Content-Type", "text/xml; charset=utf-8")

	c := appengine.NewContext(req)
	q := datastore.NewQuery("Post").Order("-Date").Limit(100)
	posts := make([]Post, 0, 100)

	if _, err := q.GetAll(c, &posts); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\r\n"))
	for _, v := range posts {
		if !strings.HasPrefix(v.Slug, "DRAFT-") {
			rw.Write([]byte(fmt.Sprintf(" <url><loc>https://blog.benjojo.co.uk/post/%s</loc> </url>\r\n", v.Slug)))
		}
	}
	rw.Write([]byte("</urlset>"))

}
