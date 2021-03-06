/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

/*

Gouplo is a simple & easy-to-use fileserver, written in Go (golang.org), that features a basic login system and a multiple-file upload form.

**features**
* jQuery/Ajax for dynamic form submissions without page reload.
* HTTP Digest Authentication for basic login capability (better than no login).
* Multiple file handling for batch upload jobs (no progress bar).
* Easily configurable with commandline flags (run 'gouplo -help' for more information).

*/
package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	auth "github.com/abbot/go-http-auth"
	"html/template"
	"io"
	"net/http"
	"os"
)

var addr = flag.String("addr", ":8080", "http service address")
var homefile = flag.String("home", "home.html", "full path to home html template file")
var uploadDir = flag.String("upload", "upload/", "full path to upload directory")
var publicDir = flag.String("public", "public/", "full path to public directory")
var createDirs = flag.Bool("create", false, "create missing directories")
var username = flag.String("user", "myuser", "set server login user name")
var password = flag.String("pass", "mypass", "set server login password")
var realm = flag.String("realm", "myrealm", "set server realm")

func homeHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	template.Must(template.ParseFiles(*homefile)).Execute(w, r.Host)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		err := r.ParseMultipartForm(100000)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		m := r.MultipartForm
		files := m.File["files"]
		for i, _ := range files {
			file, err := files[i].Open() //open file
			defer file.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			dst, err := os.Create(*uploadDir + files[i].Filename) //ensure destination is writeable
			defer dst.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if _, err := io.Copy(dst, file); err != nil { //write the file to destination
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		io.WriteString(w, "successful")
	}
}

func CalculateHA1(user, rlm, pass string) string {
	h := md5.New()
	fmt.Fprintf(h, "%s:%s:%s", user, rlm, pass)
	ha1 := h.Sum(nil)
	return fmt.Sprintf("%x", ha1)
}

func Secret(user, rlm string) string {
	if user == *username {
		return CalculateHA1(user, rlm, *password)
	}
	return ""
}

func init() {
	flag.Parse()
	if *createDirs == true {
		if err := os.Mkdir(*uploadDir, 0777); err == nil {
			fmt.Println("directory created: ", *uploadDir)
		}
		if err := os.Mkdir(*publicDir, 0777); err == nil {
			fmt.Println("directory created: ", *publicDir)
		}
	}
}

func main() {
	authenticator := auth.NewDigestAuthenticator(*realm, Secret)
	http.HandleFunc("/", authenticator.Wrap(homeHandler))
	http.HandleFunc("/upload", uploadHandler)
	http.Handle("/pub/", http.StripPrefix("/pub/", http.FileServer(http.Dir(*publicDir))))
	http.ListenAndServe(*addr, nil)
}
