package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type proxyCheckResult struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status int    `json:"status,omitempty"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

type proxyCheckResponse struct {
	Proxy struct {
		Disabled     bool   `json:"disabled"`
		YouTubeProxy string `json:"youtube_proxy"`
		HTTPProxy    string `json:"http_proxy"`
	} `json:"proxy"`
	Checks []proxyCheckResult `json:"checks"`
	AllOK  bool               `json:"all_ok"`
}

func (h *Handler) ProxyCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userAgent := resolveUserAgent()
	proxyCfg := resolveProxyConfig(h.Logger)
	logProxySettings(h.Logger, proxyCfg)

	client := &http.Client{
		Transport: headerRoundTripper{
			base: proxyCfg.baseTransport,
			headers: http.Header{
				"User-Agent": []string{userAgent},
			},
		},
		Timeout: 15 * time.Second,
	}

	targets := []struct {
		name string
		url  string
	}{
		{name: "youtube_generate_204", url: "https://www.youtube.com/generate_204"},
		{name: "googlevideo_generate_204", url: "https://redirector.googlevideo.com/generate_204"},
	}

	results := make([]proxyCheckResult, 0, len(targets))
	allOK := true

	for _, target := range targets {
		result := proxyCheckResult{
			Name: target.name,
			URL:  target.url,
		}

		req, err := http.NewRequest(http.MethodGet, target.url, nil)
		if err != nil {
			result.OK = false
			result.Error = err.Error()
			allOK = false
			results = append(results, result)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			result.OK = false
			result.Error = err.Error()
			allOK = false
			results = append(results, result)
			continue
		}

		result.Status = resp.StatusCode
		result.OK = resp.StatusCode >= 200 && resp.StatusCode < 400
		if !result.OK {
			allOK = false
		}

		_ = resp.Body.Close()
		results = append(results, result)
	}

	var response proxyCheckResponse
	response.Proxy.Disabled = proxyCfg.proxyDisabled
	response.Proxy.YouTubeProxy = redactProxyURL(proxyCfg.proxyURLStr)
	response.Proxy.HTTPProxy = redactProxyURL(proxyCfg.httpProxyEnv)
	response.Checks = results
	response.AllOK = allOK

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
