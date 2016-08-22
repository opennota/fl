// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
// Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

const (
	torHost   = "flibustahezeous3.onion"
	i2pHost   = "flibusta.i2p"
	i2pBase32 = "zmw2cyw2vj7f6obx3msmdvdepdhnw2ctc4okza2zjxlukkdfckhq.b32.i2p"
)

var (
	torAddr  = flag.String("tor", "127.0.0.1:9050", "Tor service address")
	i2pAddr  = flag.String("i2p", "127.0.0.1:4444", "I2P service address")
	forceI2P = flag.Bool("force-i2p", false, "Force I2P")
	addr     = flag.String("http", ":1338", "HTTP service address")

	host string

	c http.Client
)

func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func logRequest(r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	log.Println(host, r.Method, r.URL, r.Referer(), r.UserAgent())
}

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func reverseProxy(w http.ResponseWriter, req *http.Request) {
	logRequest(req)

	outReq := new(http.Request)
	outReq.Method = req.Method
	outReq.URL = &url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	outReq.Proto = "HTTP/1.1"
	outReq.ProtoMajor = 1
	outReq.ProtoMinor = 1
	outReq.Header = make(http.Header)
	outReq.Body = req.Body
	outReq.ContentLength = req.ContentLength
	outReq.Host = host

	for _, h := range hopHeaders {
		req.Header.Del(h)
	}
	copyHeader(outReq.Header, req.Header)
	outReq.Header.Set("Host", host)
	outReq.Header.Set("Referer", host)
	outReq.Header.Set("Origin", host)

	resp, err := c.Transport.RoundTrip(outReq)
	if err != nil {
		log.Printf("proxy error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	reqHost, _, err := net.SplitHostPort(req.Host)
	for _, cookie := range cookies {
		if err == nil {
			cookie.Domain = reqHost
		} else {
			cookie.Domain = req.Host
		}
		resp.Header.Add("Set-Cookie", cookie.String())
	}

	for _, h := range hopHeaders {
		resp.Header.Del(h)
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		if u, err := url.Parse(loc); err == nil && (u.Host == host || u.Host == i2pBase32) {
			u.Scheme = "http"
			u.Host = req.Host
			resp.Header.Set("Location", u.String())
		}
	}
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func robotsHandler(w http.ResponseWriter, req *http.Request) {
	logRequest(req)
	w.Write([]byte("User-agent: *\nDisallow: /\n"))
}

func acceptsConnections(addr string) bool {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func main() {
	flag.Parse()

	if !*forceI2P && acceptsConnections(*torAddr) {
		dialer, err := proxy.SOCKS5("tcp", *torAddr, nil, proxy.Direct)
		if err != nil {
			log.Fatalf("can't connect to Tor: %v", err)
		}
		c.Transport = &http.Transport{
			Dial: dialer.Dial,
		}
		host = torHost
		log.Println("using Tor at", *torAddr)
	} else {
		proxyURL := url.URL{
			Scheme: "http",
			Host:   *i2pAddr,
		}
		c.Transport = &http.Transport{
			Proxy: http.ProxyURL(&proxyURL),
		}
		host = i2pHost
		log.Println("using I2P at", *i2pAddr)
	}

	http.HandleFunc("/", reverseProxy)
	http.HandleFunc("/robots.txt", robotsHandler)
	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
