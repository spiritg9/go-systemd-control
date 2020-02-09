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

	//http.Post("http://localhost:8080", "application/json", bytes.NewBuffer(postCmdBytes))

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

	case "POST":
		/*b, err := ioutil.ReadAll(r.Body)
		var postCmd = PostCommand{}
		err = json.Unmarshal(b, &postCmd)
		if err != nil {
			log.Fatal(err)
		}

		out, err := ExecSystemCtl(postCmd)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(w, "%s\n", out)*/
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
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

		//fmt.Fprintf(w, "%s", b)
		//fmt.Fprintf(w, "%s", resp)

		/*for _, s := range services {
			fmt.Fprintf(w, "%v", s)
		}*/
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

}

func main() {

	http.HandleFunc("/list", systemdlist)
	http.HandleFunc("/status", systemdstatus)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
