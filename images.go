package main

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
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
				<div class="content-img">
					<img style="max-width: 100%;" src="/static/images/{{ .Hash }}{{ .Ext }}" />
				</div>
				<div class="content-edit">
					<h5>Edit Tags:</h5>
					<form action="/images/update/{{ .Hash }}" method="post">
						<textarea name="tags">{{ range $tag, $element := .Tags }}{{ $tag }} {{ end }}</textarea>
						<input type="submit" value="Save changes" />
					</form>
				</div>
			</div>
		</div>
		<footer>
			Footer
		</footer>
	</body>
</html>
`))

func parseTags(tagQuery string) (include, exclude []string) {
	for _, tag := range strings.Fields(tagQuery) {
		tag = strings.ToLower(tag)
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
	db.mu.RLock()
	urls, err := db.lookupByTags(parseTags(searchTags))
	db.mu.RUnlock()
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
	db.mu.RLock()
	entry, ok := db.Images[ps.ByName("img")]
	db.mu.RUnlock()
	if !ok {
		http.NotFound(w, req)
		return
	}
	showImageTemplate.Execute(w, entry)
}

func (db *imageDB) imageUpdateHandlerPOST(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	tags, badTags := parseTags(req.FormValue("tags"))
	if len(tags) == 0 {
		http.Error(w, "failed to update image: please supply at least one tag", http.StatusBadRequest)
		return
	} else if len(badTags) != 0 {
		http.Error(w, "failed to update image: tags may not begin with a -", http.StatusBadRequest)
		return
	}
	hash := ps.ByName("img")
	// remove the image, then re-add it with the new tag set.
	db.mu.Lock()
	defer db.mu.Unlock()
	err := db.setTags(hash, tags)
	if err != nil {
		http.Error(w, "Update failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	db.save()

	http.Redirect(w, req, "/images/show/"+hash, http.StatusSeeOther)
}

func (db *imageDB) imageDeleteHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	db.mu.Lock()
	defer db.mu.Unlock()
	entry := db.Images[ps.ByName("img")]
	err := db.removeImage(entry.Hash)
	if err != nil {
		http.Error(w, "Delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	db.save()
	// delete image + thumbnail from disk
	os.Remove(filepath.Join("static", "images", entry.Hash+entry.Ext))
	os.Remove(filepath.Join("static", "thumbnails", entry.Hash+".jpg"))

	http.Redirect(w, req, "/images", http.StatusSeeOther)
}
