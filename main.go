package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

func main() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	var err error
	ytURL := r.URL.Query().Get("url")
	if ytURL != "" {
		ytURL, err = url.QueryUnescape(ytURL)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	log.Println("query url:", ytURL)
	if ytURL == "" {
		log.Println("index")
		f, err := os.Open("./index.html")
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	} else {
		br := r.URL.Query().Get("br")
		if br == "" {
			br = "128"
		}
		var buf *bytes.Buffer
		cmdYoutubeDl := exec.Command("pipenv", "run", "youtube-dl", "-e", ytURL)
		cmdYoutubeDl.Stdout = buf
		cmdYoutubeDl.Stderr = os.Stderr
		err := cmdYoutubeDl.Run()
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		title := buf.String()
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url.PathEscape(title))
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		log.Println("Download from " + ytURL)
		cmdYoutubeDl = exec.Command("pipenv", "run", "youtube-dl", ytURL, "-f", "bestaudio", "-o", "-")
		pReader, pWriter, _ := os.Pipe()
		cmdYoutubeDl.Stdout = pWriter
		cmdYoutubeDl.Stderr = os.Stderr
		cmdFfmpeg := exec.Command("ffmpeg", "-i", "pipe:", "-f", "mp3", "-b:a", br+"k", "-")
		cmdFfmpeg.Stdin = pReader
		cmdFfmpeg.Stderr = os.Stderr
		cmdFfmpeg.Stdout = w
		cmdYoutubeDl.Start()
		cmdFfmpeg.Start()
		cmdYoutubeDl.Wait()
		pWriter.Close()
		cmdFfmpeg.Wait()
	}
}
