// Package plugindemo a demo plugin.
package ms_redirections

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/patrickmn/go-cache"
)

// Config the plugin configuration.
type Config struct {
	Headers map[string]string `json:"headers,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Headers: make(map[string]string),
	}
}

// Demo a Demo plugin.
type Demo struct {
	next     http.Handler
	headers  map[string]string
	name     string
	template *template.Template
	cache    *cache.Cache
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Headers) == 0 {
		return nil, fmt.Errorf("headers cannot be empty")
	}

	return &Demo{
		headers:  config.Headers,
		next:     next,
		name:     name,
		template: template.New("demo").Delims("[[", "]]"),
		cache:    cache.New(5*time.Minute, 10*time.Minute),
	}, nil
}

// populate a map with redirect rules
func (a *Demo) Populate() {
	fmt.Println("Populate called")
}

func (a *Demo) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// os.Stdout.WriteString("\nServeHTTP called\n")
	start := time.Now()

	for key, value := range a.headers {
		tmpl, err := a.template.Parse(value)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		writer := &bytes.Buffer{}

		err = tmpl.Execute(writer, req)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		req.Header.Set(key, writer.String())
	}

	var host = req.URL.Host
	if len(host) == 0 {
		host = req.Host
	}
	var path = req.URL.Path

	// var message = fmt.Sprintf("Found host %s and path %s \n", host, path)
	// os.Stdout.WriteString(message)

	response, err := http.Get("http://redirection-service:8086/?host=" + host + "&url=" + path)

	if err != nil {
		// just continue
		os.Stdout.WriteString("Error calling redirection service\n")
	} else {
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)

		if err != nil {
			os.Stdout.WriteString("Error reading response from redirection service\n")
		} else {
			// os.Stdout.WriteString("Response from redirection service\n")
			// os.Stdout.WriteString(string(body) + "\n")

			// if status 200 - do redirect
			// if status 404 - do nothing
			elapsed := time.Since(start)
			var message = fmt.Sprintf("%s %s : %s : took %s\n", host, path, string(body), elapsed)
			os.Stdout.WriteString(message)
			if response.StatusCode == 200 {
				http.Redirect(rw, req, string(body), http.StatusMovedPermanently)
				return
			}

			// os.Stdout.WriteString( + "\n")
			// os.Stdout.WriteString( + "\n")
		}
	}

	// http.Redirect(rw, req, "https://www.disney.com", http.StatusMovedPermanently)
	// rw.Header().Set("X-FREDRIK", "TEST")

	a.next.ServeHTTP(rw, req)
}
