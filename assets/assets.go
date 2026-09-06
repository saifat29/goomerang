package assets

import _ "embed"

//go:embed GeoLite2-City.mmdb
var GeoIPDB []byte
