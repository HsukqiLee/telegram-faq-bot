package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
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
		InsecureSkipVerify: true,
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
}

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

// DoRequest 执行带有反检测特性的HTTP请求
func DoRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	// 设置真实的浏览器头部
	setRealisticHeaders(req, "json")

	// 添加随机延迟
	addRandomDelay()

	// 执行请求并重试
	return cdo(client, req)
}

// DoRequestWithCompression 执行带压缩支持的请求
func DoRequestWithCompression(client *http.Client, req *http.Request) (*http.Response, error) {
	// 设置真实的浏览器头部，但不包含accept-encoding以避免压缩问题
	setRealisticHeaders(req, "json")

	// 移除自动压缩以避免解析问题
	req.Header.Del("accept-encoding")

	// 添加随机延迟
	addRandomDelay()

	// 执行请求并重试
	return cdo(client, req)
}

// cdo 带重试机制的请求执行
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
		// 在重试前增加延迟
		if i < 2 {
			time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)
		}
	}
	return nil, err
}

func generateEdgeUserAgent() string {
	edgeVersion := rand.Intn(5) + 136
	chromiumVersion := edgeVersion

	return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.0.0 Safari/537.36 Edg/%d.0.0.0",
		chromiumVersion, edgeVersion)
}

func generateSecChUA() string {
	edgeVersion := rand.Intn(5) + 136
	chromiumVersion := edgeVersion
	notBrandVersion := rand.Intn(10) + 20

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
	return languages[rand.Intn(len(languages))]
}

func addRandomDelay() {
	if rand.Intn(10) == 0 {
		delay := time.Duration(rand.Intn(100)+50) * time.Millisecond
		time.Sleep(delay)
	}
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

	// 添加一些常见的浏览器头部
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("connection", "keep-alive")

	// 为API请求添加适当的Origin和Referer
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
	ClientSessionHeaders.DNT = strconv.Itoa(rand.Intn(2))
}

// GetEnhancedClient 获取增强的HTTP客户端
func GetEnhancedClient() *http.Client {
	return EnhancedHttpClient
}

// GetEnhancedClientWithTimeout 获取带自定义超时的增强HTTP客户端
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

// init 初始化随机种子和会话头部
func init() {
	rand.Seed(time.Now().UnixNano())
	ResetSessionHeaders()
}
