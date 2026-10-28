// Command setup obtains and verifies a fixed AUTH V2 refresh token for Polochon.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/odwrtw/polochon/modules/simkl"
)

func run() error {
	clientID := flag.String("client-id", "", "Simkl AUTH V2 client ID")
	timeout := flag.Duration("timeout", 10*time.Minute, "maximum time to wait for approval (at most 15m)")
	flag.Parse()
	if *clientID == "" {
		return fmt.Errorf("-client-id is required")
	}
	if *timeout <= 0 || *timeout > 15*time.Minute {
		return fmt.Errorf("-timeout must be between 0 and 15m")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	// Confidential registrations can provide a secret without putting it in argv.
	secret := os.Getenv("SIMKL_CLIENT_SECRET")
	token, err := simkl.AuthorizeDevice(ctx, simkl.Params{ClientID: *clientID, ClientSecret: secret}, func(d simkl.DeviceAuthorization) error {
		link := d.VerificationURIComplete
		if link == "" {
			link = d.VerificationURI
		}
		_, err := fmt.Fprintf(os.Stderr, "Open %s\nCode: %s\nWaiting for approval and verifying two cold-start refreshes…\n", link, d.UserCode)
		return err
	})
	if err != nil {
		return err
	}
	entry := map[string]string{"name": "simkl", "client_id": *clientID, "refresh_token": token}
	if secret != "" {
		entry["client_secret"] = secret
	}
	data, err := yaml.Marshal(map[string]any{"modules_params": []map[string]string{entry}})
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Verified. Copy the following into your protected configuration. Use a separate grant for each running instance.")
	_, err = os.Stdout.Write(data)
	return err
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
