package geoip

import (
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"

	"github.com/oschwald/maxminddb-golang"
)

// GeoIP provides local IP -> ISO 3166-1 alpha-2 country code ("CN" / "US") lookups,
// backed by a MaxMind GeoLite2-Country.mmdb database.
//
// Design: Load() once at startup; failure is non-fatal — every subsequent lookup
// returns an empty string and callers treat the region as "unknown", so business
// logic (and frontend gating that defaults to allow) is never blocked.
type GeoIP struct {
	reader atomic.Pointer[maxminddb.Reader]
	loaded atomic.Bool
}

var (
	defaultGeo = &GeoIP{}
	loadOnce   sync.Once
)

// Default returns the process-wide singleton.
func Default() *GeoIP { return defaultGeo }

// Load opens the mmdb file. The path is resolved from the argument, then the
// GEOIP_DB_PATH env var, then a set of common fallback locations.
func (g *GeoIP) Load(path string) error {
	if path == "" {
		path = os.Getenv("GEOIP_DB_PATH")
	}
	if path == "" {
		for _, p := range []string{"/app/geo/GeoLite2-Country.mmdb", "geo/GeoLite2-Country.mmdb", "../geo/GeoLite2-Country.mmdb"} {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}
	if path == "" {
		return os.ErrNotExist
	}
	reader, err := maxminddb.Open(path)
	if err != nil {
		return err
	}
	old := g.reader.Swap(reader)
	if old != nil {
		_ = old.Close()
	}
	g.loaded.Store(true)
	return nil
}

// EnsureLoaded tries to load the database once; failure only logs a warning.
func EnsureLoaded() {
	loadOnce.Do(func() {
		if err := defaultGeo.Load(""); err != nil {
			common.SysLog("[geo] GeoLite2 database not loaded: " + err.Error() + " (region features fall back to unknown)")
		} else {
			common.SysLog("[geo] GeoLite2 database loaded")
		}
	})
}

// IsLoaded reports whether the database loaded successfully.
func (g *GeoIP) IsLoaded() bool { return g.loaded.Load() }

// CountryCode returns the ISO country code for an IP; empty on failure, private IP, or when unloaded.
func (g *GeoIP) CountryCode(ipStr string) string {
	if g == nil {
		return ""
	}
	reader := g.reader.Load()
	if reader == nil {
		return ""
	}
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return ""
	}
	// X-Forwarded-For may be "ip1, ip2, ip3" — take the first.
	if idx := strings.Index(ipStr, ","); idx >= 0 {
		ipStr = strings.TrimSpace(ipStr[:idx])
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	if isPrivateIP(ip) {
		return ""
	}
	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	if err := reader.Lookup(ip, &record); err != nil {
		return ""
	}
	return strings.ToUpper(record.Country.ISOCode)
}

// CountryCode is the package-level convenience wrapper over the singleton.
func CountryCode(ipStr string) string { return defaultGeo.CountryCode(ipStr) }

// isPrivateIP detects RFC1918 / loopback / link-local addresses so local/dev
// requests are not mislabeled as a specific country.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		case ip4[0] == 127:
			return true
		case ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127:
			return true
		}
	}
	if ip.To4() == nil {
		if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
			return true
		}
	}
	return false
}
