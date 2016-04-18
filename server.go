package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

var port = flag.String("port", ":3000", "port the server will listen on")

func indexHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	http.Redirect(w, req, "/images", http.StatusMovedPermanently)
}

func main() {
	flag.Parse()

	// open image DB
	imgDB, err := newImageDB("imagedb.json")
	if err != nil {
		log.Fatal(err)
		return
	}

	// ensure we have image+thumbnail+queue directories
	dirs := []string{"static/images", "static/thumbnails", "queue"}
	for _, d := range dirs {
		err = os.MkdirAll(d, 0700)
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/images", imgDB.imageSearchHandler)
	router.GET("/images/upload", imgDB.imageUploadHandler)
	router.POST("/images/upload", imgDB.imageUploadHandlerPOST)
	router.POST("/images/update/:img", imgDB.imageUpdateHandlerPOST)
	router.GET("/images/delete/:img", imgDB.imageDeleteHandler)
	router.GET("/images/show/:img", imgDB.imageShowHandler)
	router.GET("/admin", imgDB.adminHandler)
	router.GET("/admin/queue", imgDB.adminQueueHandler)
	router.POST("/admin/queue", imgDB.adminQueueHandlerPOST)
	router.GET("/admin/queue/:path", imgDB.adminQueueImg)

	router.ServeFiles("/static/*filepath", http.Dir("static"))

	log.Println("Listening...")
	err = http.ListenAndServe(*port, router)
	if err != nil {
		log.Println(err)
	}
}
