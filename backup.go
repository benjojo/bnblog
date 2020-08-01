package main

import (
	"archive/tar"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

func Producebackup(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/x-tar")

	c := appengine.NewContext(req)
	q := datastore.NewQuery("Post").Order("-Date").Limit(100)
	posts := make([]Post, 0, 100)

	tw := tar.NewWriter(rw)

	if _, err := q.GetAll(c, &posts); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, v := range posts {
		if !strings.HasPrefix(v.Slug, "DRAFT-") {
			postd, _ := base64.StdEncoding.DecodeString(v.Content)
			newpost := PostFormatted{
				Author:  v.Author,
				Content: string(postd),
				Date:    v.Date.Format("2006-01-02 15:04:05"),
				Slug:    v.Slug,
				Title:   v.Title,
			}

			hdr := &tar.Header{
				Name: fmt.Sprintf("%s.md", v.Slug),
				Size: int64(len(postd)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
				//log.Fatalln(err)
			}
			if _, err := tw.Write([]byte(postd)); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
				//log.Fatalln(err)
			}

			idk, _ := json.Marshal(newpost)
			hdr = &tar.Header{
				Name: fmt.Sprintf("%s_meta.json", v.Slug),
				Size: int64(len(idk)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
				//log.Fatalln(err)
			}
			if _, err := tw.Write([]byte(idk)); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
				//log.Fatalln(err)
			}
		}
	}

	if req.URL.Query().Get("no-files") == "" {
		filePipe := make(chan backupFile)
		go ExportAllFiles(rw, req, filePipe)

		for v := range filePipe {

			FileName := fmt.Sprintf("files/%s.blob", v.Name)
			MimeType, err := mime.ExtensionsByType(v.Type)
			if err == nil {
				if len(MimeType) != 0 {
					FileName = fmt.Sprintf("files/%s%s", v.Name, MimeType[0])
				}
			}

			hdr := &tar.Header{
				Name: FileName,
				Size: int64(v.Size),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}

			if _, err := io.Copy(tw, v.Reader); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tw.Close(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
