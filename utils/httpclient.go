package utils

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

var (
	UA_Browser = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36 Edg/137.0.0.0"

	ClientSessionHeaders = &SessionHeaders{
		UserAgent:      "",
		SecChUA:        "",
		AcceptLanguage: "",
		DNT:            "0",
	}
)

// secureRandInt 生成安全的随机整数，范围 [0, max)
func secureRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// 如果加密随机数生成失败，返回0作为后备
		return 0
	}
	return int(n.Int64())
}

var Dialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

var ClientProxy = http.ProxyFromEnvironment

func UseLastResponse(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

type CustomTransport struct {
	Dialer   *net.Dialer
	Resolver *net.Resolver
	Network  string
	Proxy    func(*http.Request) (*url.URL, error)
	Base     *http.Transport
}

func (t *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if t.Resolver != nil {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := t.Resolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			var filteredIPs []net.IP
			for _, ip := range ips {
				if (t.Network == "tcp4" && ip.To4() != nil) || (t.Network == "tcp6" && ip.To4() == nil) || t.Network == "tcp" {
					filteredIPs = append(filteredIPs, ip)
				}
			}
			for _, ip := range filteredIPs {
				ipAddr := net.JoinHostPort(ip.String(), port)
				conn, err := t.Dialer.DialContext(ctx, t.Network, ipAddr)
				if err == nil {
					return conn, nil
				}
			}
			return nil, fmt.Errorf("failed to connect to any resolved IP addresses for %s", addr)
		}
		return t.Dialer.DialContext(ctx, t.Network, addr)
	}
	t.Base.DialContext = dialContext
	t.Base.Proxy = t.Proxy
	if err := http2.ConfigureTransport(t.Base); err == nil {
		t.Base.ForceAttemptHTTP2 = true
	}
	return t.Base.RoundTrip(req)
}

func createEdgeTLSConfig() *tls.Config {
	spec, _ := utls.UTLSIdToSpec(utls.HelloEdge_Auto)

	return &tls.Config{
		// 注意：InsecureSkipVerify 设置为 true 是为了模拟某些浏览器行为
		// 在生产环境中，请根据具体需求考虑设置为 false 以增强安全性
		InsecureSkipVerify: false,            // 修改为 false 以增强安全性
		MinVersion:         tls.VersionTLS12, // 强制使用 TLS 1.2 或更高版本
		MaxVersion:         spec.TLSVersMax,
		CipherSuites:       spec.CipherSuites,
		ClientSessionCache: tls.NewLRUClientSessionCache(32),
		NextProtos:         []string{"h2", "http/1.1"},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
		},
		PreferServerCipherSuites: false,
		SessionTicketsDisabled:   false,
	}
}

/* createInsecureTLSConfig 创建一个不验证证书的TLS配置（仅在特殊情况下使用）
func createInsecureTLSConfig() *tls.Config {
	spec, _ := utls.UTLSIdToSpec(utls.HelloEdge_Auto)

	return &tls.Config{
		InsecureSkipVerify: true, // 仅在特殊情况下使用
		MinVersion:         spec.TLSVersMin,
		MaxVersion:         spec.TLSVersMax,
		CipherSuites:       spec.CipherSuites,
		ClientSessionCache: tls.NewLRUClientSessionCache(32),
		NextProtos:         []string{"h2", "http/1.1"},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
		},
		PreferServerCipherSuites: false,
		SessionTicketsDisabled:   false,
	}
}*/

var tlsConfig = createEdgeTLSConfig()

var AutoTransport = &CustomTransport{
	Dialer:   Dialer,
	Resolver: Dialer.Resolver,
	Network:  "tcp",
	Proxy:    ClientProxy,
	Base: &http.Transport{
		MaxIdleConns:           100,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    30 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		TLSClientConfig:        tlsConfig,
		MaxResponseHeaderBytes: 262144,
	},
}

var EnhancedHttpClient = &http.Client{
	Timeout:       30 * time.Second,
	CheckRedirect: UseLastResponse,
	Transport:     AutoTransport,
}

type H [2]string

func DoRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	setRealisticHeaders(req, "json")
	return cdo(client, req)
}

func DoRequestWithCompression(client *http.Client, req *http.Request) (*http.Response, error) {
	setRealisticHeaders(req, "json")
	req.Header.Del("accept-encoding")
	return cdo(client, req)
}

func cdo(c *http.Client, req *http.Request) (resp *http.Response, err error) {
	deadline := time.Now().Add(30 * time.Second)
	for i := 0; i < 3; i++ {
		if time.Now().After(deadline) {
			break
		}
		if resp, err = c.Do(req); err == nil {
			return resp, nil
		}
		if strings.Contains(err.Error(), "no such host") {
			break
		}
		if strings.Contains(err.Error(), "timeout") {
			break
		}
		if i < 2 {
			time.Sleep(time.Duration(secureRandInt(1000)+500) * time.Millisecond)
		}
	}
	return nil, err
}

func generateEdgeUserAgent() string {
	edgeVersion := secureRandInt(5) + 136
	chromiumVersion := edgeVersion

	return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.0.0 Safari/537.36 Edg/%d.0.0.0",
		chromiumVersion, edgeVersion)
}

func generateSecChUA() string {
	edgeVersion := secureRandInt(5) + 136
	chromiumVersion := edgeVersion
	notBrandVersion := secureRandInt(10) + 20

	return fmt.Sprintf(`"Microsoft Edge";v="%d", "Chromium";v="%d", "Not/A)Brand";v="%d"`,
		edgeVersion, chromiumVersion, notBrandVersion)
}

func getRandomAcceptLanguage() string {
	languages := []string{
		"en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7",
		"en-US,en;q=0.9",
		"zh-CN,zh;q=0.9,en;q=0.8",
		"zh-CN,zh;q=0.9",
		"en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,ja;q=0.6",
	}
	return languages[secureRandInt(len(languages))]
}

func setRealisticHeaders(req *http.Request, requestType string) {
	if ClientSessionHeaders.UserAgent == "" {
		ResetSessionHeaders()
	}
	req.Header.Set("user-agent", ClientSessionHeaders.UserAgent)

	switch requestType {
	case "json":
		req.Header.Set("accept", "application/json, text/plain, */*")
		req.Header.Set("content-type", "application/json")
	case "html":
		req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	default:
		req.Header.Set("accept", "*/*")
	}

	req.Header.Set("sec-ch-ua", ClientSessionHeaders.SecChUA)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("accept-language", ClientSessionHeaders.AcceptLanguage)
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("sec-fetch-site", "cross-site")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("dnt", ClientSessionHeaders.DNT)
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("connection", "keep-alive")

	if requestType == "json" {
		host := req.URL.Host
		scheme := req.URL.Scheme
		if scheme == "" {
			scheme = "https"
		}
		origin := fmt.Sprintf("%s://%s", scheme, host)
		req.Header.Set("origin", origin)
		req.Header.Set("referer", origin+"/")
	}
}

type SessionHeaders struct {
	UserAgent      string
	SecChUA        string
	AcceptLanguage string
	DNT            string
}

func ResetSessionHeaders() {
	ClientSessionHeaders.UserAgent = generateEdgeUserAgent()
	ClientSessionHeaders.SecChUA = generateSecChUA()
	ClientSessionHeaders.AcceptLanguage = getRandomAcceptLanguage()
	ClientSessionHeaders.DNT = strconv.Itoa(secureRandInt(2))
}

func GetEnhancedClientWithTimeout(timeout time.Duration) *http.Client {
	transport := &CustomTransport{
		Dialer:   Dialer,
		Resolver: Dialer.Resolver,
		Network:  "tcp",
		Proxy:    ClientProxy,
		Base: &http.Transport{
			MaxIdleConns:           100,
			IdleConnTimeout:        90 * time.Second,
			TLSHandshakeTimeout:    30 * time.Second,
			ExpectContinueTimeout:  1 * time.Second,
			TLSClientConfig:        tlsConfig,
			MaxResponseHeaderBytes: 262144,
		},
	}

	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: UseLastResponse,
		Transport:     transport,
	}
}
