package service

import (
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/proxyurl"
)

const (
	FallbackModeNone              = "none"
	FallbackModeProxy             = "proxy"
	FallbackModeDirect            = "direct"
	ProxyConnectionModeReuse      = "reuse"
	ProxyConnectionModePerRequest = "per_request"
	ProxyPerRequestURLFragment    = proxyurl.PerRequestFragment
)

type Proxy struct {
	ID             int64
	Name           string
	Protocol       string
	ConnectionMode string
	Host           string
	Port           int
	Username       string
	Password       string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      *time.Time
	FallbackMode   string
	BackupProxyID  *int64
	ExpiryWarnDays int
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

func validProxyConnectionMode(protocol, mode string) bool {
	if mode == "" || mode == ProxyConnectionModeReuse {
		return true
	}
	return mode == ProxyConnectionModePerRequest && (protocol == "http" || protocol == "https")
}

// IsExpired 报告代理是否已过期（基于 expires_at，与 status 无关）。
func (p *Proxy) IsExpired(now time.Time) bool {
	return p.ExpiresAt != nil && !p.ExpiresAt.After(now)
}

func (p *Proxy) URL() string {
	u := &url.URL{
		Scheme: p.Protocol,
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
	}
	if p.Username != "" && p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	if p.ConnectionMode == ProxyConnectionModePerRequest {
		u.Fragment = ProxyPerRequestURLFragment
	}
	return u.String()
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64
	LatencyMs      *int64
	LatencyStatus  string
	LatencyMessage string
	IPAddress      string
	Country        string
	CountryCode    string
	Region         string
	City           string
	QualityStatus  string
	QualityScore   *int
	QualityGrade   string
	QualitySummary string
	QualityChecked *int64
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Notes    *string
}
