package main

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

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
			<a href="/images">Dispel</a>
		</header>
		<div class="flex">
			<div class="upload-form">
				<form enctype="multipart/form-data" action="/images/upload" method="post">
					<div>
						<input type="file" name="image" id="upload-input" style="max-width: 100%;" />
					</div>
					<div>
						<input type="text" placeholder="Or, paste a URL" name="url" id="link-input" />
					</div>
					<div>
						<input type="text" placeholder="Add some tags" name="tags" id="user-tags" />
					</div>
					<div>
						<input type="submit" value="Upload Image" />
					</div>
				</form>
			</div>
			<div class="upload-preview">
				<img id="preview-img" style="max-width: 100%;" />
			</div>
		</div>
		<footer></footer>
	</body>
	<script>
		// load a locally-uploaded image
		document.getElementById("upload-input").onchange = function(g) {
			// clear the tag + URL fields
			document.getElementById("user-tags").value = "";
			document.getElementById("link-input").value = "";

			var reader = new FileReader();

			reader.onload = function (e) {
				document.getElementById("preview-img").src = e.target.result;
			};

			// read the image file as a data URL.
			reader.readAsDataURL(this.files[0]);
		};

		// load an image from a URL
		document.getElementById("link-input").onkeyup = function(e) {
			// clear the tag + file fields
			document.getElementById("user-tags").value = "";
			document.getElementById("upload-input").value = "";

			document.getElementById("preview-img").src = e.target.value;
		};
	</script>
</html>
`))

func (db *imageDB) imageUploadHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
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

	// image may be local or from URL
	var file io.ReadCloser
	var ext string
	if url := req.FormValue("url"); url != "" {
		resp, err := http.Get(url)
		if err != nil {
			http.Error(w, "failed to retrieve URL: "+err.Error(), http.StatusInternalServerError)
			return
		}
		file = resp.Body
		ext = filepath.Ext(url)
	} else {
		formFile, header, err := req.FormFile("image")
		if err != nil {
			http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		exts, err := mime.ExtensionsByType(header.Header.Get("Content-Type"))
		if err != nil || len(exts) == 0 {
			http.Error(w, "failed to read uploaded image data: unrecognized MIME type: "+header.Header.Get("Content-Type"), http.StatusInternalServerError)
			return
		}
		file = formFile
		ext = exts[0]
	}
	defer file.Close()

	// add to database
	hash, err := db.Upload(file, tags, ext)
	if err != nil {
		http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/images/show/"+hash, http.StatusSeeOther)
}
