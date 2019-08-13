package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	ytURL := r.URL.Query().Get("url")
	if ytURL == "" {
		log.Println("index")
		t, err := template.ParseFiles("./index.html")
		if err != nil {
			log.Println(err)
		}
		index := template.Must(t, err)
		// index.Execute(w, nil)
		err = index.Execute(w, nil)
		if err != nil {
			log.Println(err)
		}
	} else if r.URL.Path == "/title" {
		d, err := goquery.NewDocument(ytURL)
		if err != nil {
			log.Print("failed to get youtube html: ", err)
			http.Error(w, "failed to copy", http.StatusInternalServerError)
		}
		var title string
		var escaped string
		s := d.Find("meta[name=\"title\"]")
		if a, ok := s.Attr("content"); ok {
			title = a
			escaped = url.PathEscape(a)
		}
		index := template.Must(template.ParseFiles("./index.html"))
		err = index.Execute(w, struct {
			Title   string
			Escaped string
			URL     string
		}{title, escaped, ytURL})
		if err != nil {
			log.Println(err)
		}
		return
	} else {
		download(w, r)
	}
}

func download(w http.ResponseWriter, r *http.Request) {
	ytURL := r.URL.Query().Get("url")
	w.Header().Set("Content-Type", "audio/mpeg")
	w.WriteHeader(http.StatusOK)
	log.Println("Download to " + " from " + ytURL)
	cmdYoutubeDl := exec.Command("pipenv", "run", "youtube-dl", ytURL, "-f", "bestaudio", "-o", "-")
	pReader, pWriter, _ := os.Pipe()
	cmdYoutubeDl.Stdout = pWriter
	cmdYoutubeDl.Stderr = os.Stderr
	cmdFfmpeg := exec.Command("ffmpeg", "-i", "pipe:", "-q:a", "1", "-y", "-f", "mp3", "-")
	cmdFfmpeg.Stdin = pReader
	cmdFfmpeg.Stderr = os.Stderr
	cmdFfmpeg.Stdout = w
	cmdYoutubeDl.Start()
	cmdFfmpeg.Start()
	cmdYoutubeDl.Wait()
	pWriter.Close()
	cmdFfmpeg.Wait()
}
