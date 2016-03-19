package main

import (
	"crypto/md5"
	"encoding/hex"
	"image"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	// register these image formats
	_ "image/gif"
	"image/jpeg" // need full import, since we write jpeg thumbnails
	_ "image/png"

	"github.com/julienschmidt/httprouter"
	"github.com/nfnt/resize"
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
			Header
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
		<footer>
			Footer
		</footer>
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

	// simultaneously copy image to disk, calculate md5 hash, and generate thumbnail
	tmpFile, err := ioutil.TempFile(os.TempDir(), "dispel")
	if err != nil {
		http.Error(w, "failed to read uploaded image data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tmpFile.Close()
	hasher := md5.New()
	img, _, err := image.Decode(
		io.TeeReader(
			file, // decode file data
			io.MultiWriter(
				tmpFile, // also write to disk
				hasher,  // also write to hasher
			),
		),
	)
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

	// create thumbnail
	thumbFile, err := os.Create(filepath.Join("static", "thumbnails", hash+".jpg"))
	if err != nil {
		http.Error(w, "failed to generate thumbnail: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer thumbFile.Close()
	thumb := resize.Thumbnail(150, 150, img, resize.MitchellNetravali)
	err = jpeg.Encode(thumbFile, thumb, nil)
	if err != nil {
		http.Error(w, "failed to generate thumbnail: "+err.Error(), http.StatusInternalServerError)
		os.Remove(filepath.Join("static", "thumbnails", hash+".jpg"))
		return
	}

	// add image to database
	date := time.Now().Format("Mon Jan 02 15:04:05 EST 2006")
	err = db.addImage(hash, ext, date, tags)
	if err != nil && err != errImageExists {
		http.Error(w, "failed to add image: "+err.Error(), http.StatusInternalServerError)
		os.Remove(filepath.Join("static", "images", hash+ext))
		os.Remove(filepath.Join("static", "thumbnails", hash+".jpg"))
		return
	}
	db.save()

	http.Redirect(w, req, "/images/show/"+hash, http.StatusSeeOther)
}
