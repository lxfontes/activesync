package main

import (
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
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

type ProfileProtocols struct {
	ActiveSync bool `json:"active_sync"`
	ImapSSL    bool `json:"imap_ssl"`
	SMTPSSL    bool `json:"smtp_ssl"`
}

type Profile struct {
	Success   int              `json:"success"`
	Protocols ProfileProtocols `json:"protocols"`
}

// Response from /emailUser/protocols/{email}
type ProfileResponse struct {
	Result Profile `json:"result"`
}

type DeviceRegistrationResult struct {
	Success   int `json:"success"`
	AccountId int `json:"account_id"`
}

// Response from /emailUser/registerDeviceActiveSync/{email}
type DeviceRegistrationResponse struct {
	Result DeviceRegistrationResult `json:"result"`
}

type Cluster struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

func (c *Cluster) RandMember() string {
	return c.Members[rand.Intn(len(c.Members))]
}

type Config struct {
	Port               int       `json:"port"`
	ProfileAPIURL      string    `json:"profile_api_url"`
	RegistrationAPIURL string    `json:"register_device_api_url"`
	Clusters           []Cluster `json:"clusters"`
}

var apiClient = &http.Client{
	// Quanto tempo a API tem pra responder antes de desistir
	Timeout: 5 * time.Second,
}

func notifyDeviceRegistration(cfg *Config, email, deviceType, deviceId, deviceName, activeSyncHost string) error {
	registrationURL := fmt.Sprintf("%s%s", cfg.RegistrationAPIURL, email)

	formData := url.Values{}
	formData.Set("device_type", deviceType)
	formData.Set("device_id", deviceId)
	formData.Set("device_name", deviceName)
	formData.Set("active_sync_host", activeSyncHost)

	resp, err := apiClient.PostForm(registrationURL, formData)
	if err != nil {
		return fmt.Errorf("device registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("device registration API returned non-200 status: %d", resp.StatusCode)
	}

	var regResp DeviceRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode device registration response: %w", err)
	}

	if regResp.Result.Success == 0 {
		return fmt.Errorf("device registration failed for user %s", email)
	}

	return nil
}

func getTargetForUser(cfg *Config, username string) (string, error) {
	profileAPIURL := fmt.Sprintf("%s%s", cfg.ProfileAPIURL, username)

	httpReq, err := http.NewRequest("GET", profileAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create profile API request: %w", err)
	}

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

	if profileResp.Result.Success == 0 {
		return "", fmt.Errorf("no profile success for user %s", username)
	}

	if !profileResp.Result.Protocols.ActiveSync {
		return "", fmt.Errorf("user %s does not have ActiveSync enabled", username)
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
			DisableKeepAlives:   false,
			MaxIdleConns:        5000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     90 * time.Second,
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

		reqCmd := req.In.FormValue("Cmd")
		// If this is a device registration, notify the profile API
		if reqCmd == "Provision" {
			deviceType := req.In.FormValue("DeviceType")
			deviceId := req.In.FormValue("DeviceId")
			deviceName := req.In.UserAgent()
			activeSyncHost, _, _ := strings.Cut(targetHost, ":")

			err := notifyDeviceRegistration(&cfg, username, deviceType, deviceId, deviceName, activeSyncHost)
			if err != nil {
				log.Printf("Device registration notification failed for user %s: %v", username, err)
			} else {
				log.Printf("Device registration notified for user %s: device_type=%s, device_id=%s, device_name=%s, active_sync_host=%s",
					username, deviceType, deviceId, deviceName, activeSyncHost)
			}
		}

		originalURL.Host = targetHost
		originalURL.Scheme = "http"
		req.Out.URL = originalURL
		// make sure z-push sees the original host header
		req.Out.Host = req.In.Host

		log.Printf("[%s] Proxying to: %s", username, originalURL.String())
	}

	mux := http.NewServeMux()

	// This endpoint simulates an authentication error response
	// It's used when we can't determine a valid target for the user
	mux.HandleFunc("/auth-error", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		http.Error(w, "Authentication Error", http.StatusUnauthorized)
	})
	mux.Handle("/", proxy)

	bindAddr := net.JoinHostPort("0.0.0.0", strconv.Itoa(cfg.Port))
	log.Printf("Starting z-push-loadbalancer proxy on %s", bindAddr)
	server := &http.Server{
		Addr:              bindAddr,
		Handler:           mux,
		IdleTimeout:       120 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		ReadHeaderTimeout: 10 * time.Second,
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
