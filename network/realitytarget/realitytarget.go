// Package realitytarget fetches and tests public REALITY handshake candidates.
package realitytarget

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxFeedBytes = 256 * 1024

var BuiltInDomains = []string{
	"www.apple.com", "addons.mozilla.org", "www.amazon.com", "aws.amazon.com",
	"www.cloudflare.com", "www.microsoft.com", "visualstudio.microsoft.com",
	"www.samsung.com", "www.nvidia.com", "www.amd.com", "www.intel.com",
	"www.sony.com", "dl.google.com", "r.bing.com", "th.bing.com",
}

type Feed struct {
	Version     int      `json:"version"`
	GeneratedAt string   `json:"generated_at"`
	Domains     []string `json:"domains"`
}

type Result struct {
	Domain      string `json:"domain"`
	Successes   int    `json:"successes"`
	Attempts    int    `json:"attempts"`
	SuccessRate int    `json:"success_rate"`
	MedianMS    int64  `json:"median_ms"`
	JitterMS    int64  `json:"jitter_ms"`
	CertExpires string `json:"cert_expires,omitempty"`
}

var blockedAddressRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func FetchFeed(ctx context.Context, feedURL string) (Feed, error) {
	var feed Feed
	client := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return feed, err
	}
	req.Header.Set("User-Agent", "diandianS-UI-reality-targets/1")
	resp, err := client.Do(req)
	if err != nil {
		return feed, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return feed, fmt.Errorf("candidate feed returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes+1))
	if err != nil {
		return feed, err
	}
	return decodeFeed(body)
}

func decodeFeed(body []byte) (Feed, error) {
	var feed Feed
	if len(body) > maxFeedBytes {
		return feed, errors.New("candidate feed is too large")
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&feed); err != nil {
		return feed, err
	}
	if feed.Version != 1 {
		return feed, fmt.Errorf("unsupported candidate feed version: %d", feed.Version)
	}
	feed.Domains = normalizeDomains(feed.Domains, 500)
	if len(feed.Domains) == 0 {
		return feed, errors.New("candidate feed has no valid domains")
	}
	return feed, nil
}

func normalizeDomains(domains []string, limit int) []string {
	seen := make(map[string]struct{}, len(domains))
	result := make([]string, 0, len(domains))
	for _, raw := range domains {
		domain := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(raw, ".")))
		if !validHostname(domain) {
			continue
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		result = append(result, domain)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func validHostname(domain string) bool {
	if len(domain) < 4 || len(domain) > 253 || !strings.Contains(domain, ".") {
		return false
	}
	if net.ParseIP(domain) != nil || strings.ContainsAny(domain, "_/:* ") {
		return false
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}

func isPublicAddress(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	for _, prefix := range blockedAddressRanges {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

func SelectBatch(domains []string, batch uint64, limit int) []string {
	selected := append([]string(nil), domains...)
	// 同一批次顺序稳定；更换批次会得到另一组候选，避免依赖进程内随机状态。
	sort.SliceStable(selected, func(i, j int) bool {
		left := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", batch, selected[i])))
		right := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", batch, selected[j])))
		return bytes.Compare(left[:], right[:]) < 0
	})
	if limit > 0 && len(selected) > limit {
		selected = selected[:limit]
	}
	return selected
}

func Scan(ctx context.Context, domains []string, attempts, workers int) []Result {
	if len(domains) == 0 || attempts < 1 || workers < 1 {
		return []Result{}
	}
	jobs := make(chan string)
	results := make(chan Result, len(domains))
	var wg sync.WaitGroup
	if len(domains) < workers {
		workers = len(domains)
	}
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range jobs {
				result := probe(ctx, net.DefaultResolver, domain, attempts)
				// 至少两次成功，避免把偶然握手成功的域名推荐给用户。
				if result.Successes >= 2 {
					results <- result
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, domain := range domains {
			select {
			case jobs <- domain:
			case <-ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	close(results)

	output := make([]Result, 0, len(domains))
	for result := range results {
		output = append(output, result)
	}
	sort.SliceStable(output, func(i, j int) bool {
		if output[i].SuccessRate != output[j].SuccessRate {
			return output[i].SuccessRate > output[j].SuccessRate
		}
		if output[i].MedianMS != output[j].MedianMS {
			return output[i].MedianMS < output[j].MedianMS
		}
		if output[i].JitterMS != output[j].JitterMS {
			return output[i].JitterMS < output[j].JitterMS
		}
		return output[i].Domain < output[j].Domain
	})
	return output
}

type ipResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

func probe(ctx context.Context, resolver ipResolver, domain string, attempts int) Result {
	result := Result{Domain: domain, Attempts: attempts}
	resolveCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	addresses, err := resolver.LookupNetIP(resolveCtx, "ip", domain)
	cancel()
	if err != nil {
		return result
	}
	publicAddresses := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		if isPublicAddress(address) {
			publicAddresses = append(publicAddresses, address.Unmap())
		}
	}
	if len(publicAddresses) == 0 {
		return result
	}

	latencies := make([]int64, 0, attempts)
	var earliestExpiry time.Time
	for attempt := range attempts {
		attemptCtx, attemptCancel := context.WithTimeout(ctx, 2*time.Second)
		started := time.Now()
		var state tls.ConnectionState
		succeeded := false
		// 没有 IPv6 路由时，先得到 AAAA 不应让整个域名误判失败。
		for offset := range len(publicAddresses) {
			address := publicAddresses[(attempt+offset)%len(publicAddresses)]
			rawConn, dialErr := (&net.Dialer{}).DialContext(attemptCtx, "tcp", net.JoinHostPort(address.String(), "443"))
			if dialErr != nil {
				continue
			}
			tlsConn := tls.Client(rawConn, &tls.Config{
				ServerName: domain, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
				NextProtos: []string{"h2"}, CurvePreferences: []tls.CurveID{tls.X25519},
			})
			handshakeErr := tlsConn.HandshakeContext(attemptCtx)
			state = tlsConn.ConnectionState()
			_ = tlsConn.Close()
			if handshakeErr == nil && state.Version == tls.VersionTLS13 && state.NegotiatedProtocol == "h2" && len(state.VerifiedChains) > 0 {
				succeeded = true
				break
			}
		}
		attemptCancel()
		if !succeeded {
			continue
		}
		latencies = append(latencies, time.Since(started).Milliseconds())
		result.Successes++
		if len(state.PeerCertificates) > 0 && (earliestExpiry.IsZero() || state.PeerCertificates[0].NotAfter.Before(earliestExpiry)) {
			earliestExpiry = state.PeerCertificates[0].NotAfter
		}
	}
	if len(latencies) == 0 {
		return result
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	result.SuccessRate = result.Successes * 100 / attempts
	result.MedianMS = latencies[len(latencies)/2]
	result.JitterMS = latencies[len(latencies)-1] - latencies[0]
	if !earliestExpiry.IsZero() {
		result.CertExpires = earliestExpiry.UTC().Format(time.RFC3339)
	}
	return result
}
