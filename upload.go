package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/julienschmidt/httprouter"
)

func (db *imageDB) imageUploadHandlerPOST(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// parse tags
	tags, badTags := parseTags(req.FormValue("tags"))
	if len(tags) == 0 {
		http.Error(w, "failed to add image: please supply at least one tag", http.StatusBadRequest)
		return
	} else if len(badTags) != 0 {
		http.Error(w, "failed to add image: tags may not begin with a -", http.StatusBadRequest)
		return
	}

	// write image to disk with md5 filename
	file, header, err := req.FormFile("image")
	if err != nil {
		http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	exts, err := mime.ExtensionsByType(header.Header.Get("Content-Type"))
	if err != nil || len(exts) == 0 {
		http.Error(w, "failed to read uploaded image data: unrecognized MIME type: "+header.Header.Get("Content-Type"), http.StatusInternalServerError)
		return
	}
	ext := exts[0]
	tmpFile, err := ioutil.TempFile(os.TempDir(), "dispel")
	if err != nil {
		http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	hasher := md5.New()
	defer tmpFile.Close()
	_, err = io.Copy(io.MultiWriter(tmpFile, hasher), file)
	if err != nil {
		http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	hash := hex.EncodeToString(hasher.Sum(nil))
	err = os.Rename(tmpFile.Name(), filepath.Join("static", "images", hash+ext))
	if err != nil {
		http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: create thumbnail

	// add image to database
	date := time.Now().Format("Mon Jan 02 15:04:05 EST 2006")
	err = db.addImage(hash, ext, date, tags)
	if err != nil && err != errImageExists {
		http.Error(w, "failed to add image: "+err.Error(), http.StatusInternalServerError)
		return
	}
	db.save()

	http.Redirect(w, req, "/images/show/"+hash, http.StatusSeeOther)
}
