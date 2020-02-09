package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
)

// TODO: replace all log.Fatal with something more appropriate

type Services struct {
	Services []Service
}

type Service struct {
	Service string
	Load    string
	Active  string
	Sub     string
	Desc    string
}

type PostCommand struct {
	Command string
	Service string
}

func systemdstatus(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/status" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	postCmd := PostCommand{Command: "status", Service: "cron.service"}
	postCmdBytes, err := json.Marshal(postCmd)
	if err != nil {
		log.Fatal(err)
	}

	switch r.Method {
	case "GET":
		resp, err := http.Post("http://localhost:8081", "application/json", bytes.NewBuffer(postCmdBytes))
		if err != nil {
			log.Fatal(err)
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(w, "%v", b)
	default:
		fmt.Fprintf(w, "Sorry, only GET method is supported.")
	}

}

func systemdaction(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/action" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "POST":
		action := r.PostFormValue("action")
		service := r.PostFormValue("service")
		fmt.Fprintf(w, "%s\n", action)
		fmt.Fprintf(w, "%s\n", service)

		postCmd := PostCommand{Command: action, Service: service}
		postCmdBytes, err := json.Marshal(postCmd)
		if err != nil {
			log.Fatal(err)
		}

		resp, err := http.Post("http://localhost:8081", "application/json", bytes.NewBuffer(postCmdBytes))
		if err != nil {
			log.Fatal(err)
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(w, "%s\n", b)
	default:
		fmt.Fprintf(w, "Sorry, method %v is not supported, only POST is supported.", r.Method)
	}

}

func systemdlist(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/list" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		resp, err := http.Get("http://localhost:8081")
		if err != nil {
			log.Fatal(err)
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		services := []Service{}

		err = json.Unmarshal(b, &services)
		if err != nil {
			log.Fatal(err)
		}

		ss := Services{}
		ss.Services = services
		tmpl := template.Must(template.ParseFiles("layout.html"))
		err = tmpl.Execute(w, ss)
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

}

func main() {

	http.HandleFunc("/list", systemdlist)
	http.HandleFunc("/status", systemdstatus)
	http.HandleFunc("/action", systemdaction)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
