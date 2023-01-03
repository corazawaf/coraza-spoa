// Copyright 2023 The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"time"
)

func logger(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dt := time.Now()

		// get client & server ip
		clientAddr, clientPort, _ := net.SplitHostPort(r.RemoteAddr)

		adr := r.Context().Value(http.LocalAddrContextKey)
		serverAddr, serverPort, _ := net.SplitHostPort(fmt.Sprintf("%v", adr))

		// Save a copy of this request for debugging.
		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Println(err)
		}

		fmt.Printf("REQUEST: %s | %s:%s -> %s:%s \n\n",
			dt.Format("2006-01-02 15:04:05"), clientAddr, clientPort, serverAddr, serverPort)
		fmt.Printf("%s\n\n--\n\n", string(requestDump))

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)

		dump, err := httputil.DumpResponse(rec.Result(), true)
		if err != nil {
			log.Println(err)
		}
		fmt.Print(string(dump))
		fmt.Print("-------------------------------------------------------------------------\n")

		// we copy the captured response headers to our new response
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		// grab the captured response body
		data := rec.Body.Bytes()

		_, err = w.Write(data)
		if err != nil {
			log.Println(err)
		}
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body := "Welcome (request not blocked by Corza)!\n\n"

		_, err := w.Write([]byte(body))
		if err != nil {
			log.Println(err)
		}
	})

	// Use this endpoint to test response blocking
	http.HandleFunc("/leak", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body := "<html><head><title>Index of /</title></head><body><h1>Index of /</h1></body></html>\n\n"

		_, err := w.Write([]byte(body))
		if err != nil {
			log.Println(err)
		}
	})

	port := ":3000"
	fmt.Println("Server is running on port: " + port)

	// Start server on port specified above
	log.Fatal(http.ListenAndServe(port, logger(http.DefaultServeMux)))
}
