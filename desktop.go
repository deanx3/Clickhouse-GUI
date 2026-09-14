//go:build desktop

package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	webview "github.com/webview/webview_go"
)

func init() { desktop = runDesktop }

// runDesktop serves the app on loopback only and shows it in a native WebKitGTK
// window. Closing the window exits the process.
func runDesktop(h http.Handler) {
	if u := os.Getenv("CHGUI_USER"); u != "" {
		c := cred{User: u, Pass: os.Getenv("CHGUI_PASSWORD"), At: time.Now().Unix()}
		// Prove the credentials once, so a typo lands on the login page with
		// something to fix instead of an app where every query fails.
		if _, err := query(conns[0], c, "SELECT 1", nil); err != nil {
			log.Printf("auto sign-in as %s failed, showing the login page: %v", u, err)
		} else {
			autoCred = &c
		}
	}

	// A stable port keeps the origin stable, which is what keeps query history
	// and preferences (localStorage) across launches.
	_, port, _ := net.SplitHostPort(env("CHGUI_ADDR", ":8090"))
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Printf("%v; using a random port, so history and preferences will not persist", err)
		if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
			log.Fatal(err)
		}
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("ClickHouse Console")
	w.SetSize(1440, 900, webview.HintNone)
	w.Navigate("http://" + ln.Addr().String())
	w.Run()
}
