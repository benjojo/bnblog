package bnblog

import (
	"fmt"

	"github.com/codegangsta/martini"

	"io/ioutil"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
	"google.golang.org/cloud/storage"
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
	storageclient, err := storage.NewClient(c)
	defer storageclient.Close()
	actualbucket := storageclient.Bucket(bucket)

	fn := RandString(10)

	bin, _ := ioutil.ReadAll(file_from_client)

	wc1 := actualbucket.Object(fn).NewWriter(c)
	wc1.ContentType = headers.Header.Get("Content-Type")

	wc1.ACL = []storage.ACLRule{{storage.AllUsers, storage.RoleReader}}
	if _, err := wc1.Write(bin); err != nil {
		log.Warningf(c, "ouch! %s", err)
	}
	if err := wc1.Close(); err != nil {
		log.Warningf(c, "ouch! %s", err)
	}
	// log.Infof(c, "updated object:", wc1.Object())

	rw.Write([]byte(fn))
	log.Warningf(c, "fin.")

}

func ReadFile(rw http.ResponseWriter, req *http.Request, params martini.Params) {
	var c context.Context
	c = appengine.NewContext(req)

	var err error
	var bucket string
	if bucket, err = file.DefaultBucketName(c); err != nil {
		// log.Errorf(c, "failed to get default GCS bucket name: %v", err)
		return
	}

	storageclient, err := storage.NewClient(c)
	defer storageclient.Close()
	actualbucket := storageclient.Bucket(bucket)

	rc, err := actualbucket.Object(params["tag"]).NewReader(c)
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
	o, err := actualbucket.Object(params["tag"]).Attrs(c)
	if err != nil {
		rw.Header().Add("Content-Type", "image/png")
	} else {
		rw.Header().Add("Content-Type", o.ContentType)
	}

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

	storageclient, _ := storage.NewClient(c)
	defer storageclient.Close()
	actualbucket := storageclient.Bucket(bucket)

	query := &storage.Query{Prefix: ""}
	for query != nil {

		objs := actualbucket.Objects(c, query)

		for {
			obj, err := objs.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return
			}

			newfile := File{}
			newfile.Name = obj.Name
			newfile.Type = obj.ContentType

			rc, err := actualbucket.Object(obj.Name).NewReader(c)
			if err != nil {
				log.Warningf(c, "readFile: unable to open file from bucket %q, file %q: %v", bucket, obj.Name, err)
				return
			}
			defer rc.Close()
			slurp, err := ioutil.ReadAll(rc)
			if err != nil {
				log.Warningf(c, "readFile: unable to read data from bucket %q, file %q: %v", bucket, obj.Name, err)
				return
			}
			newfile.Content = slurp
			export = append(export, newfile)
		}
	}

	return export
}

func GimmeDC(rw http.ResponseWriter, req *http.Request) string {
	c := appengine.NewContext(req)
	return appengine.Datacenter(c)
}
