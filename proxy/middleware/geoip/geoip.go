package geoip

import (
	"net/http"
	"net/netip"
	"strconv"

	"github.com/oschwald/maxminddb-golang/v2"
	"github.com/rs/zerolog/log"
	"github.com/tomasen/realip"

	"github.com/saifat29/goomerang/proxy/middleware"
)

const (
	headerCountry     = "X-GeoIP-Country"
	headerCountryName = "X-GeoIP-Country-Name"
	headerCity        = "X-GeoIP-City"
	headerRegion      = "X-GeoIP-Region"
	headerLatitude    = "X-GeoIP-Latitude"
	headerLongitude   = "X-GeoIP-Longitude"
)

// record represents the relevant fields from a GeoLite2-City database record.
type record struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
}

// New returns a middleware that looks up the client IP in a MaxMind GeoIP
// database and injects geoip data in request headers.
func New(db *maxminddb.Reader) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP, err := netip.ParseAddr(realip.FromRequest(r))
			if err != nil {
				log.Debug().Err(err).Str("ip", clientIP.String()).Msg("invalid ip")
				next.ServeHTTP(w, r)
				return
			}

			var rec record
			if err := db.Lookup(clientIP).Decode(&rec); err != nil {
				log.Debug().Err(err).Str("ip", clientIP.String()).Msg("geoip lookup failed")
				next.ServeHTTP(w, r)
				return
			}

			injectHeaders(w.Header(), rec)
			next.ServeHTTP(w, r)
		})
	}
}

// injectHeaders sets X-GeoIP-* headers on the response writer for non-empty fields.
func injectHeaders(header http.Header, rec record) {
	if rec.Country.ISOCode != "" {
		header.Set(headerCountry, rec.Country.ISOCode)
	}

	if name, ok := rec.Country.Names["en"]; ok && name != "" {
		header.Set(headerCountryName, name)
	}

	if name, ok := rec.City.Names["en"]; ok && name != "" {
		header.Set(headerCity, name)
	}

	if len(rec.Subdivisions) > 0 {
		if name, ok := rec.Subdivisions[0].Names["en"]; ok && name != "" {
			header.Set(headerRegion, name)
		}
	}

	if rec.Location.Latitude != 0 {
		header.Set(headerLatitude, strconv.FormatFloat(rec.Location.Latitude, 'f', -1, 64))
	}

	if rec.Location.Longitude != 0 {
		header.Set(headerLongitude, strconv.FormatFloat(rec.Location.Longitude, 'f', -1, 64))
	}
}
