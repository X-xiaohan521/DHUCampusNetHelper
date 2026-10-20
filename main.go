package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

var (
	username = os.Getenv("DHU_USERNAME")
	password = os.Getenv("DHU_PASSWORD")
	iface    = os.Getenv("DHU_WAN_IFACE")
)

// 版本信息，构建时通过 -ldflags "-X" 注入，未注入时保持默认值。
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

const (
	// probeInterval is how often the session is checked while it is healthy.
	probeInterval = 30 * time.Second

	// Login retry backoff: starts at backoffInitial and doubles on every
	// consecutive failure up to backoffMax.
	backoffInitial = 30 * time.Second
	backoffMax     = 5 * time.Minute
)

func main() {
	// -- prepare to login --
	// extract env variables, panic if not set
	if username == "" || password == "" || iface == "" {
		log.Fatal("Enviroment variable DHU_USERNAME, DHU_PASSWORD or DHU_WAN_IFACE not set.")
	}

	// get the ipaddr of the wan interface
	localAddr, err := getLocalAddr(iface)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Sending requests using local addr: %s\n", localAddr.IP)

	// create local dialer and transport
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		LocalAddr: &net.TCPAddr{
			IP:   localAddr.IP,
			Port: 0,
		},
		DualStack: true, // support IPv4/IPv6 simultaneously
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// create cookie jar for saving cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("Failed to create cookie jar:", err)
	}

	// client acts like a browser
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			log.Printf("Redirecting to: %s (hop %d)\n", req.URL, len(via))
			if req.Response != nil {
				log.Println("Got cookies:", req.Response.Cookies())
			}
			return nil
		},
		Transport: transport,
	}

// -- supervise the session --
// Detect whether the campus net login session has expired every probeInterval, reconnect if expired.
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("campus-net-helper %s (commit %s, built %s)\n", version, commit, buildTime)
		os.Exit(0)
	}

	backoff := backoffInitial
	for {
		expired, err := sessionExpired(client)
		switch {
		case err != nil:
			// Could not determine the state (e.g. link down).
			// Do not attempt a login, just try again on the next tick.
			log.Printf("Could not determine session state (retrying in %s): %v\n", probeInterval, err)
		case !expired:
			// Session is working properly.
			log.Printf("Session is alive.\n")
			backoff = backoffInitial
		default:
			log.Println("Session appears expired (captive portal detected), logging in...")
			if err := loginOnce(client); err != nil {
				// Backoff if login failed. Initially backoffInitial, then multiply by 2 every time it fialed (capped by backoffMax).
				log.Printf("Login failed: %v (next attempt in %s)\n", err, backoff)
				time.Sleep(backoff)
				backoff *= 2
				if backoff > backoffMax {
					backoff = backoffMax
				}
				continue
			}
			log.Println("Login succeeded, session renewed.")
			backoff = backoffInitial
		}
		time.Sleep(probeInterval)
	}
}

// sessionExpired probes the connectivity check endpoint to decide whether the
// login session has expired. It returns (true, nil) when the captive portal is
// intercepting traffic, (false, nil) when the network is genuinely reachable,
// and (false, err) when the state could not be determined at all (for example
// the link is down), in which case the caller should not attempt a login.
func sessionExpired(client *http.Client) (expired bool, err error) {
	req, err := http.NewRequest("GET", "http://www.msftconnecttest.com/connecttest.txt", nil)
	if err != nil {
		expired = false
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 Edg/140.0.0.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,zh-CN;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		expired = false
		return
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	// Healthy only when the real NCSI body is returned by the real host.
	if resp.StatusCode == http.StatusOK && strings.TrimSpace(string(b)) == "Microsoft Connect Test" {
		return false, nil
	}

	// Anything else (redirect to the portal, portal page, non-200) means the session is no longer authorized.
	log.Printf("Probe indicates captive portal: status=%s host=%s\n", resp.Status, resp.Request.URL.Host)
	return true, nil
}

// loginOnce performs a single full login against the campus portal.
func loginOnce(client *http.Client) error {
	// -- start to login --
	// 1. GET login page (targeting PHPSESSID storing in cookies)
	// 1.1 access any wan address and follow redirect to the login page
	req1, err := http.NewRequest("GET", "http://www.msftconnecttest.com/redirect", nil)
	if err != nil {
		return err
	}
	// 1.2 IMPORTANT!! Must set UA to simulate a browser
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 Edg/140.0.0.0")
	req1.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req1.Header.Set("Accept-Language", "en-US,zh-CN;q=0.9")
	log.Println("Getting login page from:", req1.URL.String())
	resp, err := client.Do(req1) // send GET request
	if err != nil {
		return fmt.Errorf("failed to get login page: %w", err)
	}
	// 1.3 process login page response
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		// return if we did not get the login page
		return fmt.Errorf("failed to get login page: %s", string(b))
	}
	log.Println("Got login page:", resp.Status)
	portalHost := resp.Request.URL.Host // extract portalHost for POST login request
	log.Println("Extracted portal host:", portalHost)
	resp.Body.Close()

	// 2. POST login request
	// 2.1 build req body "username=***&password=***"
	buf := bytes.NewBuffer([]byte("username=" + username + "&" + "password=" + password))
	// 2.2 build req with portalHost
	req2, err := http.NewRequest("POST", "https://"+portalHost+"/post.php", buf)
	if err != nil {
		return err
	}
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 Edg/140.0.0.0")
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req2.Header.Set("Accept-Language", "en-US,zh-CN;q=0.9")
	log.Println("Sending login request to", req2.URL.String(), ": username="+username, "& password="+strings.Repeat("*", len(password)))
	// 2.3 send POST request
	resp, err = client.Do(req2)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	log.Println("Got login response:", resp.Status)
	b, _ = io.ReadAll(resp.Body)
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html" {
		log.Println("Login response content:", string(b))
	}
	resp.Body.Close()

	// 3. TEST conn status
	resp, err = http.Get("https://www.bilibili.com")
	if err != nil {
		return fmt.Errorf("failed to connect to bilibili: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("failed to connect to bilibili with code: %d", resp.StatusCode)
	}
	log.Println("Connection check passed.")
	resp.Body.Close()
	return nil
}

// Gets the local ip address of the network interface with the specified name.
// The addr returned will contain the address on field IP, and field Port will be 0.
func getLocalAddr(networkInterface string) (addr net.IPAddr, err error) {
	interfaces, err := net.Interfaces()

	// loop through the ifaces and find the first one whose name contains networkInterface
	for _, iface := range interfaces {
		if strings.Contains(iface.Name, networkInterface) {
			// extract all addrs on the ifacce
			addrs, e := iface.Addrs()
			if e != nil {
				log.Printf("failed to get the addrs on iface %s: %v\n", iface.Name, e)
				continue
			}

			// find the first ipv4 address and return
			for _, a := range addrs {
				if ipNet, ok := a.(*net.IPNet); ok && ipNet.IP.To4() != nil {
					addr = net.IPAddr{
						IP: ipNet.IP,
					}
					err = nil
					return
				}
			}
		}
	}

	err = fmt.Errorf("failed to extract ip on interface: %s\n", networkInterface)
	return
}
