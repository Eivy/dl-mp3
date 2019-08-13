package main

import (
	"bytes"
	"io"
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
		f, err := os.Open("./index.html")
		if err != nil {
			log.Print("failed to read: ")
			log.Println(err)
			http.Error(w, "failed to read", http.StatusInternalServerError)
		}
		_, err = io.Copy(w, f)
		if err != nil {
			log.Print("failed to copy: ")
			log.Println(err)
			http.Error(w, "failed to copy", http.StatusInternalServerError)
		}
	} else if r.URL.Path == "/title" {
		d, err := goquery.NewDocument(ytURL)
		if err != nil {
			log.Print("failed to get youtube html: ", err)
			http.Error(w, "failed to copy", http.StatusInternalServerError)
		}
		s := d.Find("meta[name=\"title\"]")
		if a, ok := s.Attr("content"); ok {
			title := url.PathEscape(a)
			w.Write([]byte(title))
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
