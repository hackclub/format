package util

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	MaxFileSize = 30 * 1024 * 1024 // 30MB
	ConnectTimeout = 10 * time.Second
	OverallTimeout = 30 * time.Second
)

// HTTPFetcher handles secure HTTP fetching with SSRF protection
type HTTPFetcher struct {
	client *http.Client
}

func NewHTTPFetcher() *HTTPFetcher {
	// Create HTTP client with timeouts and custom dialer for SSRF protection
	dialer := &net.Dialer{
		Timeout: ConnectTimeout,
	}
	
	// Custom dialer to prevent SSRF attacks
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			
			// Resolve the host to check if it's a private IP
			ips, err := net.LookupIP(host)
			if err != nil {
				return nil, err
			}
			
			// Check if any resolved IP is private/internal
			for _, ip := range ips {
				if isPrivateIP(ip) {
					return nil, fmt.Errorf("connection to private IP address is not allowed: %s", ip)
				}
			}
			
			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:    10,
		IdleConnTimeout: 90 * time.Second,
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   OverallTimeout,
	}
	
	return &HTTPFetcher{client: client}
}

func (f *HTTPFetcher) FetchURL(ctx context.Context, urlStr string) ([]byte, string, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %v", err)
	}
	
	// Only allow HTTPS
	if parsedURL.Scheme != "https" {
		return nil, "", fmt.Errorf("only HTTPS URLs are allowed")
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %v", err)
	}
	
	// Set user agent
	req.Header.Set("User-Agent", "format.hackclub.com/1.0")
	
	// Make request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	// Check content length
	if resp.ContentLength > MaxFileSize {
		return nil, "", fmt.Errorf("file too large: %d bytes (max %d)", resp.ContentLength, MaxFileSize)
	}
	
	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxFileSize))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %v", err)
	}
	
	// Get content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = DetectContentType(body)
	}
	
	return body, contentType, nil
}

// isPrivateIP checks if an IP address is in a private/internal range
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	
	// Check for private IPv4 ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	
	// Check for private IPv6 ranges
	if ip.To4() == nil {
		// fc00::/7 (unique local)
		if len(ip) >= 1 && (ip[0]&0xfe) == 0xfc {
			return true
		}
		// fe80::/10 (link-local)
		if len(ip) >= 2 && ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
			return true
		}
	}
	
	return false
}
