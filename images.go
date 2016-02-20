package main

import (
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var searchImageTemplate = template.Must(template.New("searchImage").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - Image Database</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<input type="search" placeholder="yeb guac" />
		<div>
			{{ range . }}
				<span class="thumb">
					<a href="/images/show/{{.}}">
						<img src="/static/images/{{.}}" />
					</a>
				</span>
			{{ end }}
		</div>
	</body>
</html>
`))

func imageSearchHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	images := []string{
		"tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg",
		"tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg",
		"tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg",
		"tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg",
		"tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg",
		"tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg", "tidus.jpg",
	}
	searchImageTemplate.Execute(w, images)
}

var showImageTemplate = template.Must(template.New("showImage").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<title>Dispel - {{.}}</title>
		<link rel="stylesheet" href="/static/css/milligram.min.css">
		<link rel="stylesheet" href="/static/css/images.css">
	</head>
	<body>
		<header>
			Header
		</header>
		<div class="flex">
			<div class="sidebar">
				Sidebar
			</div>
			<div class="content">
				<img style="max-width: 100%;" src="/static/images/{{.}}" />
			</div>
		</div>
		<footer>
			Footer
		</footer>
	</body>
</html>
`))

func imageShowHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// TODO: look up img
	showImageTemplate.Execute(w, ps.ByName("img"))
}
