// Command provisioners creates and deletes dedicated Stormkit Cloud instances
// by hand, for operators and for testing the flow end to end. Only Hetzner
// Cloud is supported for now.
//
//	go run ./src/cloud/provisioners/cmd create -name acme [-email owner@acme.com] [-domain acme.example.com]
//	go run ./src/cloud/provisioners/cmd delete -id 12345678
//
// Configuration is read from the environment; see hetzner.ConfigFromEnv.
// create prints the instance as JSON, including its owner API key, which is
// not recoverable afterwards.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/stormkit-io/stormkit-io/src/cloud/provisioners"
	"github.com/stormkit-io/stormkit-io/src/cloud/provisioners/hetzner"
)

const usage = `usage:
  provisioners create -name <name> [-email <email>] [-domain <domain>]
  provisioners delete -id <server-id>`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}

	cfg, err := hetzner.ConfigFromEnv()

	if err != nil {
		return err
	}

	var p provisioners.Provisioner = hetzner.New(cfg)

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("create", flag.ContinueOnError)
		name := fs.String("name", "", "instance name (lowercase letters, digits, dashes)")
		email := fs.String("email", "", "owner admin email (default agent@<domain>)")
		domain := fs.String("domain", "", "instance base domain (default <ip>.sslip.io)")

		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "creating instance %q, this takes several minutes...\n", *name)

		instance, err := p.CreateInstance(ctx, provisioners.CreateInstanceParams{
			Name:       *name,
			AdminEmail: *email,
			Domain:     *domain,
		})

		if err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(instance)

	case "delete":
		fs := flag.NewFlagSet("delete", flag.ContinueOnError)
		id := fs.Int64("id", 0, "Hetzner server ID")

		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		if *id <= 0 {
			return errors.New("-id is required")
		}

		if err := p.DeleteInstance(ctx, *id); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "deleted server %d\n", *id)

		return nil

	default:
		return errors.New(usage)
	}
}
