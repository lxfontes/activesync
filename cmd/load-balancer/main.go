package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/elazarl/goproxy"
)

func setRequestError(req *http.Request, errorPath string) {
	req.URL.Path = errorPath
	req.URL.Scheme = "http"
	req.URL.Host = "localhost"
}

func hashUsername(username string, buckets int) int {
	var hash int
	for _, char := range username {
		hash += int(char)
	}
	return (hash % buckets)
}

type Cluster struct {
	ID      string
	Members []string
	Scheme  string
}

func (c *Cluster) RandMember() string {
	return c.Members[rand.Intn(len(c.Members))]
}

func main() {

	clusters := []*Cluster{
		{
			ID:      "a",
			Members: []string{"z-push-a-1:8080", "z-push-a-2:8080"},
			Scheme:  "http",
		},
		{
			ID:      "b",
			Members: []string{"z-push-b-1:8080"},
			Scheme:  "http",
		},
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.KeepDestinationHeaders = true
	proxy.KeepHeader = true
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(goproxy.UrlHasPrefix("/")).DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			username, _, ok := req.BasicAuth()
			if !ok {
				// if we don't have any credentials, then this is likely an autodiscover request
				// send it anywhere
				selectedCluster := clusters[rand.Intn(len(clusters))]
				targetHost := selectedCluster.RandMember()

				req.URL.Host = targetHost
				req.URL.Scheme = selectedCluster.Scheme

				log.Printf("[unauthenticated] Proxying to: %s", req.URL.String())
				return req, nil
			}

			clusterIdx := hashUsername(username, len(clusters))
			selectedCluster := clusters[clusterIdx]

			targetHost := selectedCluster.RandMember()

			req.URL.Host = targetHost
			req.URL.Scheme = selectedCluster.Scheme

			log.Printf("[%s] Proxying to: %s", username, req.URL.String())
			return req, nil
		},
	)

	log.Printf("Starting z-push-loadbalancer proxy on :80")

	server := &http.Server{
		Addr:         ":80",
		Handler:      proxy,
		IdleTimeout:  30 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
