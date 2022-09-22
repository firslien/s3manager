package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
	//"log"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool) http.HandlerFunc {
	type objectWithIcon struct {
		Info minio.ObjectInfo
		Icon string
		Name string
		HumanBytesSize string
		IsDirectory bool
	}

	type pageData struct {
		BucketName  string
		BackPrefix  string
		CurrentPath string
		Objects     []objectWithIcon
		AllowDelete bool
	}

	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		var objs []objectWithIcon
		doneCh := make(chan struct{})
		defer close(doneCh)
		
		var prefix string
    	values := r.URL.Query()
    	prefix=values.Get("prefix")

		opts := minio.ListObjectsOptions{
			Recursive: listRecursive,
			Prefix: prefix,
		}

		objectCh := s3.ListObjects(r.Context(), bucketName, opts)
		for object := range objectCh {
			if object.Err != nil {
				handleHTTPError(w, fmt.Errorf("error listing objects: %w", object.Err))
				return
			}

			obj := objectWithIcon{Info: object, Icon: icon(object.Key), Name: filepath.Base(object.Key), IsDirectory: strings.HasSuffix(object.Key,"/"), HumanBytesSize: HumanBytesLoaded(object.Size)}
			
			if obj.IsDirectory {
				obj.Name = obj.Name + "/"
			}

			timeZone, _ := time.LoadLocation("Asia/Shanghai")
			obj.Info.LastModified = obj.Info.LastModified.In(timeZone)
			objs = append(objs, obj)
		}

		data := pageData{
			BucketName:  bucketName,
			BackPrefix:  filepath.Dir(filepath.Dir(prefix)) + "/",
			CurrentPath: filepath.Base(prefix),
			Objects:     objs,
			AllowDelete: allowDelete,
		}

		t, err := template.ParseFS(templates, "layout.html.tmpl", "bucket.html.tmpl")
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing template files: %w", err))
			return
		}
		err = t.ExecuteTemplate(w, "layout", data)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error executing template: %w", err))
			return
		}
	}
}

// func GetUrlArg(r *http.Request, name string) string {
//     var arg string
//     values := r.URL.Query()
//     arg=values.Get(name)
//     return arg
// }

// icon returns an icon for a file type.
func icon(fileName string) string {
	e := path.Ext(fileName)
	switch e {
	case ".tgz", ".gz", ".zip":
		return "archive"
	case ".png", ".jpg", ".gif", ".svg":
		return "photo"
	case ".mp3", ".wav":
		return "music_note"
	}

	return "insert_drive_file"
}

func  HumanBytesLoaded(s int64) string {
	if ( s < 1024 ){
		return fmt.Sprintf("%d ", s)
	}

	var suffix string;
	var b float32;
	if s > (1 << 30) {
		suffix = "G"
		b = float32(s) / (1 << 30)
	} else if s > (1 << 20) {
		suffix = "M"
		b = float32(s) / (1 << 20)
	} else if s > (1 << 10) {
		suffix = "K"
		b = float32(s) / (1 << 10)
	}
	return fmt.Sprintf("%.2f %s", b, suffix)
}