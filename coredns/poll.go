package fritzbox

import (
	"context"
	"net"
	"regexp"
	"time"

	"github.com/miekg/dns"
	"github.com/nitram509/gofritz/pkg/soap"
	"github.com/nitram509/gofritz/pkg/tr064/lan"
	"github.com/nitram509/gofritz/pkg/tr064model"
)

var (
	exprMac = regexp.MustCompile(`PC(-[A-Z0-9]{1,2}){6}`)
	exprIp4 = regexp.MustCompile(`PC(-\d{1,3}){4}`)
	exprIp6 = regexp.MustCompile(`PC(-[a-z0-9]{0,4}){6}`)
)


func poll(ctx context.Context, box Config, state Network) {
	session := soap.NewSession(box.addr, box.user, box.pass)

	tick := time.Tick(time.Minute)
	seen := make(map[string]*tr064model.XAvmGetHostListResponse)
	for {
		list, err := lan.XAvmGetHostList(session)
		if err != nil {
			log.Errorf("%s: X_AVM-DE_GetHostList: %s", box.addr, err)
			continue
		}

		clear(seen)
		for i, host := range list {
			name := host.HostName
			other, ok := seen[name]

			switch {
			case name == "fritz.box":
			case ok && other.Active:
			case exprMac.MatchString(name):
			case exprIp4.MatchString(name):
			case exprIp6.MatchString(name):
			default:
				seen[name] = &list[i]
			}
		}

		state.Lock()
		clear(state.Hosts)
		for _, host := range seen {
			ip := net.ParseIP(host.IPAddress)
			state.Hosts[clean(host.HostName)] = ip
		}
		state.Unlock()

		select {
		case <-tick:
		case <-ctx.Done():
		}
	}
}

func clean(name string) Host {
	return Host(dns.CanonicalName(name))
}
