package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

var adminQueueTemplate = template.Must(template.New("adminQueue").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - Admin Queue</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<header>
			<a href="/images">Dispel</a>
		</header>
		<div class="flex">
		</div>
		<div class="imagelist">
			{{ range $index, $entry := . }}
				<a href="/admin/queue?item={{ $index }}">
					<span class="thumb">
						{{ if eq $entry.Action "upload" }}
						<img class="preview" src="/admin/queue/{{ $entry.Hash }}_thumb.jpg" />
						{{ else }}
						<img class="preview" src="/static/thumbnails/{{ $entry.Hash }}.jpg" />
						{{ end }}
					</span>
				</a>
			{{ else }}
				<span>Nothing in the queue!</span><br/><br/>
			{{ end }}
		</div>
		<footer></footer>
	</body>
	<script>
	</script>
</html>
`))

var adminQueueDeleteTemplate = template.Must(template.New("adminQueueDelete").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - Admin Queue</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<header>
			<a href="/images">Dispel</a>
			|
			<a href="/admin/queue">Queue</a>
		</header>
		<div class="content">
			<div class="content-img">
				<img style="max-width: 100%;" src="/static/images/{{ .Hash }}{{ .Ext }}" />
			</div>
			<div class="judge">
				<h5>Delete this image?</h5>
				<form action="" method="post">
					<button type="submit" formaction="/admin/queue?item=0&approve=true">Approve</button>
					<button type="submit" formaction="/admin/queue?item=0&approve=false">Deny</button>
				</form>
			</div>
		</div>
		<footer></footer>
	</body>
	<script>
	</script>
</html>
`))

type setTagsArgs struct {
	queueItem
	Added, Removed []string
}

var adminQueueSetTagsTemplate = template.Must(template.New("adminQueueSetTags").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - Admin Queue</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<header>
			<a href="/images">Dispel</a>
			|
			<a href="/admin/queue">Queue</a>
		</header>
		<div class="content">
			<div class="content-img">
				<img style="max-width: 100%;" src="/static/images/{{ .Hash }}{{ .Ext }}" />
			</div>
			<textarea name="tags">{{ range $tag, $_ := .Tags }}{{ $tag }} {{ end }}</textarea>
			<h6>Added: <span style="color: green">{{ range .Added }}{{ . }} {{ end }}</span></h6>
			<h6>Removed: <span style="color: red">{{ range .Removed }}{{ . }} {{ end }}</span></h6>
			<div class="judge">
				<h5>Modify this image's tags?</h5>
				<form action="" method="post">
					<button type="submit" formaction="/admin/queue?item=0&approve=true">Approve</button>
					<button type="submit" formaction="/admin/queue?item=0&approve=false">Deny</button>
				</form>
			</div>
		</div>
		<footer></footer>
	</body>
	<script>
	</script>
</html>
`))

var adminQueueUploadTemplate = template.Must(template.New("adminQueueUpload").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - Admin Queue</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<header>
			<a href="/images">Dispel</a>
			|
			<a href="/admin/queue">Queue</a>
		</header>
		<div class="content">
			<div class="content-img">
				<img style="max-width: 100%;" src="/admin/queue/{{ .Hash }}{{ .Ext }}" />
			</div>
			<textarea name="tags">{{ range $tag, $_ := .Tags }}{{ $tag }} {{ end }}</textarea>
			<div class="judge">
				<h5>Add this image?</h5>
				<form action="" method="post">
					<button type="submit" formaction="/admin/queue?item=0&approve=true">Approve</button>
					<button type="submit" formaction="/admin/queue?item=0&approve=false">Deny</button>
				</form>
			</div>
		</div>
		<footer></footer>
	</body>
	<script>
	</script>
</html>
`))

func (db *imageDB) adminHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	http.Redirect(w, req, "/admin/queue", http.StatusSeeOther)
}

func (db *imageDB) adminQueueImg(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	http.ServeFile(w, req, filepath.Join("queue", ps.ByName("path")))
}

func (db *imageDB) adminQueueHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if req.FormValue("item") == "" {
		adminQueueTemplate.Execute(w, db.Queue)
		return
	}

	var index int
	_, err := fmt.Sscan(req.FormValue("item"), &index)
	if err != nil || index < 0 || index >= len(db.Queue) {
		http.Error(w, "invalid queue index", http.StatusBadRequest)
		return
	}
	switch item := db.Queue[index]; item.Action {
	case actionDelete:
		adminQueueDeleteTemplate.Execute(w, item)
	case actionSetTags:
		added, removed := db.Images[item.Hash].Tags.diff(item.Tags)
		adminQueueSetTagsTemplate.Execute(w, setTagsArgs{item, added, removed})
	case actionUpload:
		adminQueueUploadTemplate.Execute(w, item)
	default:
		http.Error(w, "unknown action: "+item.Action, http.StatusInternalServerError)
	}
}

func (db *imageDB) runDelete(item queueItem) error {
	err := db.removeImage(item.Hash)
	if err != nil {
		return err
	}
	// delete image + thumbnail from disk
	os.Remove(filepath.Join("static", "images", item.Hash+item.Ext))
	os.Remove(filepath.Join("static", "thumbnails", item.Hash+".jpg"))
	return nil
}

func (db *imageDB) runSetTags(item queueItem) error {
	_, ok := db.Images[item.Hash]
	if !ok {
		return errImageNotExists
	}
	err := db.removeImage(item.Hash)
	if err != nil {
		return err
	}
	return db.addImage(item.imageEntry)
}

func (db *imageDB) runUpload(item queueItem) error {
	// move image+thumbnail to static dir
	err := os.Rename(
		filepath.Join("queue", item.Hash+item.Ext),
		filepath.Join("static", "images", item.Hash+item.Ext),
	)
	if err != nil {
		return err
	}
	err = os.Rename(
		filepath.Join("queue", item.Hash+"_thumb.jpg"),
		filepath.Join("static", "thumbnails", item.Hash+".jpg"),
	)
	if err != nil {
		return err
	}

	// add image to database
	err = db.addImage(item.imageEntry)
	if err != nil && err != errImageExists {
		os.Remove(filepath.Join("static", "images", item.Hash+item.Ext))
		os.Remove(filepath.Join("static", "thumbnails", item.Hash+".jpg"))
		return err
	}
	return nil
}

// adminQueueHandlerPOST approves or denies an item in the queue.
func (db *imageDB) adminQueueHandlerPOST(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var approve bool
	var index int
	_, err := fmt.Sscan(req.FormValue("approve"), &approve)
	if err != nil {
		http.Error(w, "invalid approve value", http.StatusBadRequest)
		return
	}
	_, err = fmt.Sscan(req.FormValue("item"), &index)
	if err != nil || index < 0 || index > len(db.Queue) {
		http.Error(w, "invalid queue index", http.StatusBadRequest)
		return
	}
	item := db.Queue[index]

	if !approve {
		// need to delete temp file
		if item.Action == actionUpload {
			os.Remove(filepath.Join("queue", item.Hash+"_thumb.jpg"))
			os.Remove(filepath.Join("queue", item.Hash+item.Ext))
		}
		goto done
	}

	switch item.Action {
	case actionDelete:
		err = db.runDelete(item)
		if err != nil {
			http.Error(w, "failed to delete image: "+err.Error(), http.StatusInternalServerError)
			return
		}
	case actionSetTags:
		err = db.runSetTags(item)
		if err != nil {
			http.Error(w, "failed to set tags: "+err.Error(), http.StatusInternalServerError)
			return
		}
	case actionUpload:
		err = db.runUpload(item)
		if err != nil {
			http.Error(w, "failed to upload image: "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		panic("bad action type: " + item.Action)
	}

done:
	// remove from queue
	db.Queue = append(db.Queue[:index], db.Queue[index+1:]...)
	db.save()

	http.Redirect(w, req, "/admin/queue", http.StatusSeeOther)
}
