package bnblog

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/codegangsta/martini"
	"net/http"
	"strings"
	"text/template"
	"time"
)

var AdminPage = template.Must(template.ParseFiles("public/admin.html"))

func PublishPost(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	c := appengine.NewContext(req)
	u := user.Current(c)
	if u == nil {
		http.Error(rw, fmt.Sprintf("wat %s", u), http.StatusForbidden)
		return
	}

	if fmt.Sprintf("%s", u) != "ben@benjojo.co.uk" && fmt.Sprintf("%s", u) != "ben@benjojo.com" {
		http.Error(rw, fmt.Sprintf("wat? %s", u), http.StatusForbidden)
		return
	}

	req.ParseForm()
	postslug := req.PostFormValue("slug")
	if postslug == "" {
		postslug = fmt.Sprintf("DRAFT-%s", RandString(10))
	}

	k := datastore.NewKey(c, "Post", postslug, 0, nil)

	postdate, err := time.Parse("2006-01-02 15:04:05", req.PostFormValue("date"))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	NP := Post{
		Author:  "Benjojo",
		Content: base64.StdEncoding.EncodeToString([]byte(req.PostFormValue("post"))),
		Date:    postdate,
		Slug:    postslug,
		Title:   strings.Split(req.PostFormValue("post"), "\n")[0],
	}
	_, err = datastore.Put(c, k, &NP)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	} else {

		http.Error(rw, fmt.Sprintf("/post/%s", postslug), http.StatusCreated)
	}
	return
}

func Admin(rw http.ResponseWriter, req *http.Request, params martini.Params) {

	layoutData := struct {
		Date string
	}{
		Date: time.Now().Format("2006-01-02 15:04:05"),
	}

	err := AdminPage.Execute(rw, layoutData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func RandString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}
