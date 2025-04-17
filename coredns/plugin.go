package fritzbox

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin(ID)

type Plugin struct {
	next    plugin.Handler
	cfg []Config
	nets    map[Domain]Network
}

type Domain string
type Host string

type Network struct {
	*sync.RWMutex
	Hosts map[Host]net.IP
}

func (p *Plugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	req := request.Request{W: w, Req: r}

	if req.QType() != dns.TypeA {
		return plugin.NextOrFailure(p.Name(), p.next, ctx, w, r)
	}

	res := new(dns.Msg)
	res.SetReply(r)
	res.Authoritative = true

	host, domain := split(req.Name())
	if domain != "." {
		if ip, ok := p.nets[domain].Host(host); ok {
			res.Answer = []dns.RR{a(host, domain, ip)}
		}
	} else {
		for domain, net := range p.nets {
			if ip, ok := net.Host(host); ok {
				res.Answer = append(res.Answer, a(host, domain, ip))
			}
		}
	}

	if len(res.Answer) > 0 {
		w.WriteMsg(res)
		return dns.RcodeSuccess, nil
	}

	return plugin.NextOrFailure(p.Name(), p.next, ctx, w, r)
}

func a(host Host, domain Domain, ip net.IP) *dns.A {
	return &dns.A{A: ip, Hdr: dns.RR_Header{Name: string(host) + string(domain), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}}
}

func (p *Plugin) Name() string {
	return ID
}

func (p *Plugin) Start(ctx context.Context) error {
	for _, box := range p.cfg {
		state := Network{Hosts: make(map[Host]net.IP), RWMutex: new(sync.RWMutex)}
		p.nets[box.domain] = state
		go poll(ctx, box, state)
	}
	return nil
}

func (net Network) Host(name Host) (net.IP, bool) {
	if net.Hosts == nil {
		return nil, false
	}

	net.RLock()
	defer net.RUnlock()
	ip, ok := net.Hosts[name]
	return ip, ok
}

func split(name string) (Host, Domain) {
	name, domain, _ := strings.Cut(name, ".")
	return Host(dns.CanonicalName(name)), Domain(dns.CanonicalName(domain))
}
