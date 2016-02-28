package main

import (
	"crypto/md5"
	"encoding/hex"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

var searchImageTemplate = template.Must(template.New("searchImage").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - Image Database</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
		<script src="/static/js/images.js"></script>
	</head>
	<body>
		<header>
			Header
		</header>
		<div style="margin: 0 1.5% 24px 1.5%;">
			<input id="searchbar" type="search" placeholder="yeb guac" value="{{ .Search }}" />
		</div>
		<div class="imagelist">
			{{ range .Images }}
				<a href="/images/show/{{ .Hash }}">
					<span class="thumb">
						<img class="preview" src="/static/thumbnails/{{ .Hash }}.jpg" />
					</span>
				</a>
			{{ else }}
				<span>No results!</span><br/><br/>
			{{ end }}
		</div>
		<footer>
			Footer
		</footer>
	</body>
</html>
`))

var showImageTemplate = template.Must(template.New("showImage").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - {{ .Hash }}{{ .Ext }}</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<header>
			Header
		</header>
		<div class="flex">
			<div class="sidebar">
				{{ range $tag, $element := .Tags }}
					<div>
						<a href="/images?t={{ $tag }}">{{ $tag }}</a>
					</div>
				{{ end }}
			</div>
			<div class="content">
				<img style="max-width: 100%;" src="/static/images/{{ .Hash }}{{ .Ext }}" />
			</div>
		</div>
		<footer>
			Footer
		</footer>
	</body>
</html>
`))

var uploadImageTemplate = template.Must(template.New("uploadImage").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - Upload Image</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<header>
			Header
		</header>
		<div class="add-image">
			<form enctype="multipart/form-data" action="/images/upload" method="post">
				<div>
					<input type="file" name="image">
				</div>
				<div>
					<input type="text" name="tags">
				</div>
				<div>
					<input type="submit" value="Upload Image" name="submit">
				</div>
			</form>
		</div>
		<footer>
			Footer
		</footer>
	</body>
</html>
`))

func parseTags(tagQuery string) (include, exclude []string) {
	for _, tag := range strings.Split(tagQuery, " ") {
		if strings.TrimPrefix(tag, "-") == "" {
			continue
		}
		if strings.HasPrefix(tag, "-") {
			exclude = append(exclude, tag[1:])
		} else {
			include = append(include, tag)
		}
	}
	return
}

// imageSearchHandler is the handler for the /images route. If
func (db *imageDB) imageSearchHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	searchTags := req.FormValue("t")
	urls, err := db.lookupByTags(parseTags(searchTags))
	if err != nil {
		http.Error(w, "Lookup failed", http.StatusInternalServerError)
		return
	}
	// for now, limit to 100 images
	if len(urls) > 100 {
		urls = urls[:100]
	}
	searchImageTemplate.Execute(w, struct {
		Search string
		Images []imageEntry
	}{searchTags, urls})
}

func (db *imageDB) imageShowHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	entry, ok := db.Images[ps.ByName("img")]
	if !ok {
		http.NotFound(w, req)
		return
	}
	showImageTemplate.Execute(w, entry)
}

func (db *imageDB) imageUploadHandlerGET(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	uploadImageTemplate.Execute(w, nil)
}

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
	err = os.Rename(tmpFile.Name(), filepath.Join("static", "foo", "images", hash+ext))
	if err != nil {
		http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: create thumbnail

	// add image to database
	date := time.Now().Format("Mon Jan 02 15:04:05 EST 2006")
	err = db.addImage(hash, ext, date, tags)
	if err == errImageExists {
		// not really the correct way to use this status code...
		http.Redirect(w, req, "/images/show/"+hash, http.StatusSeeOther)
	} else if err != nil {
		http.Error(w, "failed to add image: "+err.Error(), http.StatusInternalServerError)
	}
}
