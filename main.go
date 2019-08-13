package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
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
		mv, err := ioutil.TempFile("/tmp", "dl-mp3")
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		mv.Close()
		os.Remove(mv.Name())
		au, err := ioutil.TempFile("/tmp", "dl-mp3")
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer au.Close()
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.(http.Flusher).Flush()
		log.Println("Download to " + mv.Name() + " from " + ytURL)
		cmd := exec.Command("pipenv", "run", "youtube-dl", "-f", "mp4", "-o", mv.Name()+".mp4", ytURL)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			log.Println("cmd err: ", err)
			log.Println("youtube-dl error output: " + stderr.String())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("Write to " + au.Name())
		cmd = exec.Command("ffmpeg", "-i", mv.Name()+".mp4", "-y", "-f", "mp3", au.Name())
		err = cmd.Run()
		if err != nil {
			log.Println("cmd err: ", err)
			log.Println("ffmpeg error output: " + stderr.String())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println("ffmpeg error output: " + stderr.String())
		b, err := ioutil.ReadAll(au)
		if err != nil {
			log.Println("failed to read: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(b)
	}
}
