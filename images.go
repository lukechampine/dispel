package main

import (
	"html/template"
	"net/http"
	"strings"

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
				<div>
					<a href="/images/delete/{{ .Hash }}">Delete</a>
				</div>
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

func (db *imageDB) imageDeleteHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	err := db.removeImage(ps.ByName("img"))
	if err != nil {
		http.Error(w, "Delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/images", http.StatusMovedPermanently)
}
