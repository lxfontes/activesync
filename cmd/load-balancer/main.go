package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	"sigs.k8s.io/yaml"
)

type ProfileRequest struct {
	Email string `json:"email"`
}

type ProfileResponse struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	// Indica se o ActiveSync está habilitado para o perfil.
	ActiveSyncEnabled bool `json:"activesync_enabled"`
	// Quando presente, indica qual host de ActiveSync usar para esse usuario ( Dedicado )
	// Quando ausente, o sistema escolhe um host.
	ActiveSyncHost string `json:"activesync_host,omitempty"`
}

type Cluster struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

func (c *Cluster) RandMember() string {
	return c.Members[rand.Intn(len(c.Members))]
}

type Config struct {
	Port          int       `json:"port"`
	ProfileAPIURL string    `json:"profile_api_url"`
	Clusters      []Cluster `json:"clusters"`
}

var apiClient = &http.Client{
	// Quanto tempo a API tem pra responder antes de desistir
	Timeout: 5 * time.Second,
}

func getTargetForUser(cfg *Config, username string) (string, error) {
	req := &ProfileRequest{
		Email: username,
	}
	profileAPIURL := cfg.ProfileAPIURL
	if profileAPIURL == "" {
		profileAPIURL = "http://localhost:8080/api/profile"
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal profile request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", profileAPIURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create profile API request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := apiClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("profile API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("profile API returned non-200 status: %d", resp.StatusCode)
	}

	var profileResp ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profileResp); err != nil {
		return "", fmt.Errorf("failed to decode profile API response: %w", err)
	}

	if !profileResp.ActiveSyncEnabled {
		return "", fmt.Errorf("user %s does not have ActiveSync enabled", username)
	}

	if profileResp.ActiveSyncHost != "" {
		// User has a dedicated host
		return profileResp.ActiveSyncHost, nil
	}

	// No dedicated host, use hash-based distribution
	clusterIdx := hashUsername(username, len(cfg.Clusters))
	selectedCluster := cfg.Clusters[clusterIdx]
	return selectedCluster.RandMember(), nil
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()
	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	errorURL, err := url.Parse(fmt.Sprintf("http://localhost:%d/auth-error", cfg.Port))
	if err != nil {
		log.Fatalf("Failed to create auth error URL: %v", err)
	}

	proxy := &httputil.ReverseProxy{
		Transport: &http.Transport{
			DisableKeepAlives: false,
			MaxIdleConns:      0,
			IdleConnTimeout:   600 * time.Second,
		},
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		// Ensure that successful responses keep the connection alive
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Header.Set("Connection", "keep-alive")
		} else {
			resp.Header.Set("Connection", "close")
		}
		return nil
	}

	proxy.Rewrite = func(req *httputil.ProxyRequest) {
		originalURL := req.In.URL
		// Set X-Real-IP headers so Z-Push can log the original client IP
		clientIP := req.In.RemoteAddr
		if xff := req.In.Header.Get("X-Forwarded-For"); xff != "" {
			clientIP = xff
			req.Out.Header.Set("X-Forwarded-For", xff)
		}
		req.Out.Header.Set("X-Real-IP", clientIP)
		req.SetXForwarded()

		// Preserve Connection header from the original request
		if connection := req.In.Header.Get("Connection"); connection != "" {
			req.Out.Header.Set("Connection", connection)
		}

		// if we don't have any credentials, then this is likely an autodiscover request
		// send it anywhere
		username, _, ok := req.In.BasicAuth()
		if !ok {
			selectedCluster := cfg.Clusters[rand.Intn(len(cfg.Clusters))]
			targetHost := selectedCluster.RandMember()

			originalURL.Host = targetHost
			originalURL.Scheme = "http"
			req.Out.URL = originalURL
			// make sure z-push sees the original host header
			req.Out.Host = req.In.Host

			log.Printf("[unauthenticated] Proxying to: %s", originalURL.String())
			return
		}

		// Regular request with username
		// Send to cluster based on hash of username
		targetHost, err := getTargetForUser(&cfg, username)
		if err != nil {
			log.Printf("Error selecting target for user %s: %v", username, err)
			req.SetURL(errorURL)
			return
		}

		originalURL.Host = targetHost
		originalURL.Scheme = "http"
		req.Out.URL = originalURL
		// make sure z-push sees the original host header
		req.Out.Host = req.In.Host

		log.Printf("[%s] Proxying to: %s", username, originalURL.String())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth-error", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		http.Error(w, "Authentication Error", http.StatusUnauthorized)
	})
	mux.Handle("/", proxy)

	bindAddr := net.JoinHostPort("0.0.0.0", strconv.Itoa(cfg.Port))
	log.Printf("Starting z-push-loadbalancer proxy on %s", bindAddr)
	server := &http.Server{
		Addr:         bindAddr,
		Handler:      mux,
		IdleTimeout:  30 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func hashUsername(username string, buckets int) int {
	var hash int
	for _, char := range username {
		hash += int(char)
	}
	return (hash % buckets)
}
