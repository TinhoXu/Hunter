package core

import (
	"context"
	"fmt"
	"github.com/Dreamacro/clash/constant"
	"github.com/TinhoXu/Hunter/common"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	NonSelfMadeAvailable = "\033[0;32m完美解锁\033[0m"        // 绿色
	SelfMadeAvailable    = "\033[0;33m仅解锁自制剧\033[0m"      // 黄色
	AreaAvailable        = "\033[0;35m仅解锁宽松版权的自制剧\033[0m" // 紫色
	NothingAvailable     = "\033[0;31m啥也不是\033[0m"        // 红色

	Available   = "\033[0;35m可以观看此影片\033[0m" // 绿色
	Unavailable = "\033[0;31m不能观看此影片\033[0m" // 红色
)

func UnlockTest(ctx context.Context, p constant.Proxy) string {
	if UnlockTestWithMovieID(ctx, p, common.NonSelfMadeAvailableID) {
		return NonSelfMadeAvailable
	} else if UnlockTestWithMovieID(ctx, p, common.SelfMadeAvailableID) {
		return SelfMadeAvailable
	} else if UnlockTestWithMovieID(ctx, p, common.AreaAvailableID) {
		return AreaAvailable
	} else {
		return NothingAvailable
	}
}

func UnlockTestWithMovieID(ctx context.Context, p constant.Proxy, movieID int) bool {
	testURL := common.NetflixUrl + strconv.Itoa(movieID)
	retCode := requestNetflixUri(ctx, p, testURL)
	return !strings.Contains(retCode, "Error") && !strings.Contains(retCode, "Ban")
}

func requestNetflixUri(ctx context.Context, p constant.Proxy, uri string) string {
	resp, err := request(ctx, p, "", uri)
	if err != nil {
		return "Error"
	}
	defer resp.Body.Close()

	Header := resp.Header

	if Header["X-Robots-Tag"] != nil {
		if Header["X-Robots-Tag"][0] == "index" {
			return "us"
		}
	}

	if Header["Location"] == nil {
		return "Ban"
	} else {
		return strings.Split(Header["Location"][0], "/")[3]
	}
}

func request(ctx context.Context, p constant.Proxy, ip, uri string) (resp *http.Response, err error) {
	// 获取 client
	addr, err := urlToMetadata(uri)
	if err != nil {
		return
	}

	instance, err := p.DialContext(ctx, &addr)
	if err != nil {
		return
	}
	defer instance.Close()

	transport := &http.Transport{
		Dial: func(string, string) (net.Conn, error) {
			return instance, nil
		},
		// from http.DefaultTransport
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()

	urlValue, err := url.Parse(uri)
	if err != nil {
		return
	}
	host := urlValue.Host

	var newUri = uri
	if ip != "" {
		newUri = strings.Replace(uri, host, ip, 1)
	}

	req, err := http.NewRequest(http.MethodGet, newUri, nil)
	req.Host = host
	req.Header.Set("USER-AGENT", common.UserAgent)

	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	return
}

func urlToMetadata(rawURL string) (addr constant.Metadata, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}

	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			err = fmt.Errorf("%s scheme not Support", rawURL)
			return
		}
	}

	addr = constant.Metadata{
		AddrType: constant.AtypDomainName,
		Host:     u.Hostname(),
		DstIP:    nil,
		DstPort:  port,
	}
	return
}
