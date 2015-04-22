package bnblog

import (
	"fmt"
	"github.com/codegangsta/martini"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/user"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
	"io/ioutil"
	"net/http"
)

func UploadFile(rw http.ResponseWriter, req *http.Request, params martini.Params) {
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

	hc := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(c, storage.ScopeFullControl),
			Base:   &urlfetch.Transport{Context: c},
		},
	}

	file_from_client, headers, err := req.FormFile("fileToUpload")

	if err != nil {
		http.Error(rw, "Was that even a file?!", http.StatusBadRequest)
		return
	}

	defer file_from_client.Close()

	bucket := ""
	if bucket == "" {
		var err error
		if bucket, err = file.DefaultBucketName(c); err != nil {
			// log.Errorf(c, "failed to get default GCS bucket name: %v", err)
			return
		}
	}
	ctx := cloud.NewContext(appengine.AppID(c), hc)

	fn := RandString(10)

	bin, _ := ioutil.ReadAll(file_from_client)

	wc1 := storage.NewWriter(ctx, bucket, fn)
	wc1.ContentType = headers.Header.Get("Content-Type")
	wc1.ACL = []storage.ACLRule{{storage.AllUsers, storage.RoleReader}}
	if _, err := wc1.Write(bin); err != nil {
		log.Warningf(c, "ouch! %s", err)
	}
	if err := wc1.Close(); err != nil {
		log.Warningf(c, "ouch! %s", err)
	}
	log.Infof(c, "updated object:", wc1.Object())

	rw.Write([]byte(fn))
	log.Warningf(c, "fin.")

}

func ReadFile(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	c := appengine.NewContext(req)

	bucket := ""
	if bucket == "" {
		var err error
		if bucket, err = file.DefaultBucketName(c); err != nil {
			// log.Errorf(c, "failed to get default GCS bucket name: %v", err)
			return
		}
	}

	hc := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(c, storage.ScopeFullControl),
			Base:   &urlfetch.Transport{Context: c},
		},
	}

	ctx := cloud.NewContext(appengine.AppID(c), hc)

	rc, err := storage.NewReader(ctx, bucket, params["tag"])
	if err != nil {
		log.Warningf(c, "readFile: unable to open file from bucket %q, file %q: %v", bucket, params["tag"], err)
		return
	}
	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		log.Warningf(c, "readFile: unable to read data from bucket %q, file %q: %v", bucket, params["tag"], err)
		return
	}
	o, _ := storage.StatObject(ctx, bucket, params["tag"])
	rw.Header().Add("Content-Type", o.ContentType)

	rw.Write([]byte(slurp))
}

type File struct {
	Name    string
	Content []byte
	Type    string
}

func ExportAllFiles(rw http.ResponseWriter, req *http.Request) (export []File) {
	c := appengine.NewContext(req)

	export = make([]File, 0)

	bucket := ""
	if bucket == "" {
		var err error
		if bucket, err = file.DefaultBucketName(c); err != nil {
			// log.Errorf(c, "failed to get default GCS bucket name: %v", err)
			return
		}
	}

	hc := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(c, storage.ScopeFullControl),
			Base:   &urlfetch.Transport{Context: c},
		},
	}

	ctx := cloud.NewContext(appengine.AppID(c), hc)

	query := &storage.Query{Prefix: ""}
	for query != nil {
		objs, err := storage.ListObjects(ctx, bucket, query)
		if err != nil {
			//d.errorf("listBucket: unable to list bucket %q: %v", bucket, err)
			return
		}
		query = objs.Next

		for _, obj := range objs.Results {
			//d.dumpStats(obj)
			newfile := File{}
			newfile.Name = obj.Name
			newfile.Type = obj.ContentType

			rc, err := storage.NewReader(ctx, bucket, obj.Name)
			if err != nil {
				log.Warningf(c, "readFile: unable to open file from bucket %q, file %q: %v", bucket, obj.Name, err)
				return
			}
			defer rc.Close()
			slurp, err := ioutil.ReadAll(rc)

			newfile.Content = slurp
			export = append(export, newfile)
		}
	}

	return export
}
