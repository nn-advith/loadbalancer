package consts

// refer RFC 7230 for why these need to be removed
// basically, these headers are reserved for hop-by-hop not end-to-end; i.e only valid between client and loadbalancer
var HopHeaders = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"TE":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

var RequiredEnvParameters = map[string]struct{}{
	"NBLB_JSONPATH": {},
	"NBLB_SCHEME":   {},
	"NBLB_STRATEGY": {},
	"NBLB_PORT":     {},
}
