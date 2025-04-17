package fritzbox

import (
	"context"
	"fmt"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

const ID = "fritzbox"

func init() {
	plugin.Register(ID, setup)
}

func setup(c *caddy.Controller) error {
	cfg := dnsserver.GetConfig(c)

	bad := func(format string, args ...any) error {
		msg := fmt.Sprintf(format, args...)
		return fmt.Errorf("%s:%d: %s", c.File(), c.Line(), msg)
	}

	p := &Plugin{
		nets: make(map[Domain]Network),
	}

	for c.Next() {
		var box Config
		var domain string
		if !c.Args(&box.addr, &domain) {
			return bad("%s:%d: expected <ip> <domain>")
		}
		box.domain = Domain(dns.CanonicalName(domain))

		for c.NextBlock() {
			switch c.Val() {
			case "user":
				c.Args(&box.user)
			case "password":
				c.Args(&box.pass)
			default:
				return bad("unknown: %s", c.Val())
			}
		}

		p.cfg = append(p.cfg, box)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.OnStartup(func() error {
		p.Start(ctx)
		return nil
	})
	c.OnShutdown(func() error {
		cancel()
		return nil
	})

	cfg.AddPlugin(func(next plugin.Handler) plugin.Handler {
		p.next = next
		return p
	})
	return nil
}

type Config struct {
	addr   string
	domain Domain

	user string
	pass string
}
