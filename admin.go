package main

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/codegangsta/martini"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

func PublishPost(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	c := appengine.NewContext(req)
	u := user.Current(c)
	if u == nil {
		http.Error(rw, fmt.Sprintf("wat %s", u), http.StatusForbidden)
		return
	}

	if !isAuthedToMakeChanges(u) {
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
		Author:       "Benjojo",
		Content:      base64.StdEncoding.EncodeToString([]byte(req.PostFormValue("post"))),
		Date:         postdate,
		Slug:         postslug,
		Title:        strings.Split(req.PostFormValue("post"), "\n")[0],
		Type:         req.PostFormValue("cata"),
		R1:           req.PostFormValue("R1"),
		R2:           req.PostFormValue("R2"),
		FeatureImage: req.PostFormValue("featureimage"),
	}
	_, err = datastore.Put(c, k, &NP)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(rw, fmt.Sprintf("/post/%s", postslug), http.StatusCreated)
	}
	forceUpdatePostTitleCache(rw, req)
	return
}

func isAuthedToMakeChanges(u *user.User) bool {
	emailHash := sha512.New()
	emailHashString := fmt.Sprintf("%x", emailHash.Sum([]byte(fmt.Sprintf("%s", u))))
	log.Printf("Email Hash Attempt %s / %s", u, emailHashString)
	if emailHashString == "8270bbd8cfc0ff367556e4ee1ec05d7f5873b65d71001c154efbb03dca232307c2616fa81ca7bacbe7e1739b8f49cf9cb4a3c3905df9faebf75992233ff4d170" {
		return true
	}
	if emailHashString == "4d98aa710d2be9770a0d173718a85dfb7b527d6c1aa7952fa3011cc2fcefa5cc22745c563faa95c5dd4a8b616f42b815f00527d4bdd033959dcd443fd3dfd8b2" {
		return true
	}
	if emailHashString == "cffd49359ad9dcadd350f9cbb913f210303d0c20c1a57eaf59c81a3301abee39377095e8ae5b827633ca7624d473361d5c37ff848418185892b7b58cca1b677a" {
		return true
	}
	return false
}

func RemovePost(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	c := appengine.NewContext(req)

	u := user.Current(c)
	if u == nil {
		http.Error(rw, fmt.Sprintf("wat %s", u), http.StatusForbidden)
		return
	}

	if !isAuthedToMakeChanges(u) {
		http.Error(rw, fmt.Sprintf("wat? %s", u), http.StatusForbidden)
		return
	}

	k := datastore.NewKey(c, "Post", params["name"], 0, nil)
	err := datastore.Delete(c, k)
	if err == nil {
		rw.Write([]byte("the deed has been done\r\n"))
	}
}

func Run_GC(rw http.ResponseWriter, req *http.Request) {
	c := appengine.NewContext(req)
	u := user.Current(c)
	if u == nil {
		http.Error(rw, fmt.Sprintf("wat %s", u), http.StatusForbidden)
		return
	}

	if !isAuthedToMakeChanges(u) {
		http.Error(rw, fmt.Sprintf("wat? %s", u), http.StatusForbidden)
		return
	}

	q := datastore.NewQuery("Post").Order("-Date").Limit(100)

	for t := q.Run(c); ; {
		var e Post
		key, err := t.Next(&e)
		if err == datastore.Done {
			break
		}
		if err != nil {
			// return nil, err
			break
		}
		if strings.HasPrefix(e.Slug, "DRAFT-") || e.Title == "" {
			// DESTROY
			err := datastore.Delete(c, key)
			if err == nil {
				rw.Write([]byte("Killed Draft\r\n"))
			}
		}
	}

}

func Admin(rw http.ResponseWriter, req *http.Request, params martini.Params) {

	layoutData := struct {
		Date string
	}{
		Date: time.Now().Format("2006-01-02 15:04:05"),
	}
	var AdminPage = template.Must(template.ParseFiles("public2/admin.html"))

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
