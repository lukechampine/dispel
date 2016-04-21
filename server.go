package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

var port = flag.String("port", ":3000", "port the server will listen on")
var adminIP = flag.String("admin", "127.0.0.1", "IP of the administrator")

func indexHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	http.Redirect(w, req, "/images", http.StatusMovedPermanently)
}

func ipWhitelist(fn httprouter.Handle, whitelistedHost string) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		if host == whitelistedHost {
			fn(w, req, ps)
		} else {
			http.Error(w, "Access denied, buttmunch. This action has been logged.", http.StatusForbidden)
			log.Printf("Unauthorized access detected: %v tried to load route %v", req.RemoteAddr, req.URL.Path)
		}
	}
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
	router.GET("/thanks", thanksHandler)
	router.GET("/images", imgDB.imageSearchHandler)
	router.GET("/images/upload", imgDB.imageUploadHandler)
	router.POST("/images/upload", imgDB.imageUploadHandlerPOST)
	router.POST("/images/update/:img", imgDB.imageUpdateHandlerPOST)
	router.GET("/images/delete/:img", imgDB.imageDeleteHandler)
	router.GET("/images/show/:img", imgDB.imageShowHandler)

	router.GET("/admin", ipWhitelist(imgDB.adminHandler, *adminIP))
	router.GET("/admin/queue", ipWhitelist(imgDB.adminQueueHandler, *adminIP))
	router.POST("/admin/queue", ipWhitelist(imgDB.adminQueueHandlerPOST, *adminIP))
	router.GET("/admin/queue/:path", ipWhitelist(imgDB.adminQueueImg, *adminIP))

	router.ServeFiles("/static/*filepath", http.Dir("static"))

	log.Println("Listening...")
	err = http.ListenAndServe(*port, router)
	if err != nil {
		log.Println(err)
	}
}
