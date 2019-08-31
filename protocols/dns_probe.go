// DNS Tester
//
// The DNS tester allows you to confirm that the specified DNS server
// returns the results you expect.  It is invoked with input like this:
//
//    ns.example.com must run dns with lookup test.example.com with type A with result '1.2.3.4'
//
// This test ensures that the DNS lookup of an A record for `test.example.com`
// returns the single value 1.2.3.4
//
// Lookups are supported for A, AAAA, MX, NS, and TXT records.
//

package protocols

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/skx/overseer/test"
)

// DNSTest is our object.
type DNSTest struct {
}

var (
	localm *dns.Msg
	localc *dns.Client
)

// lookup will perform a DNS query, using the servername-specified.
// It returns an array of maps of the response.
func (s *DNSTest) lookup(server string, name string, ltype string, timeout time.Duration) ([]string, error) {

	var results []string

	var err error
	localm = &dns.Msg{
		MsgHdr: dns.MsgHdr{
			RecursionDesired: true,
		},
		Question: make([]dns.Question, 1),
	}
	localc = &dns.Client{
		ReadTimeout: timeout,
	}
	r, err := s.localQuery(server, dns.Fqdn(name), ltype)
	if err != nil || r == nil {
		return nil, err
	}
	if r.Rcode == dns.RcodeNameError {
		return nil, fmt.Errorf("no such domain %s", dns.Fqdn(name))
	}

	for _, entry := range r.Answer {

		//
		// Lookup the value
		//
		switch ent := entry.(type) {
		case *dns.A:
			a := ent.A
			results = append(results, a.String())
		case *dns.AAAA:
			aaaa := ent.AAAA
			results = append(results, aaaa.String())
		case *dns.MX:
			mxName := ent.Mx
			mxPrio := ent.Preference
			results = append(results, fmt.Sprintf("%d %s", mxPrio, mxName))
		case *dns.NS:
			nameserver := ent.Ns
			results = append(results, nameserver)
		case *dns.TXT:
			txt := ent.Txt
			results = append(results, txt[0])
		}
	}
	return results, nil
}

// Given a name & type to lookup perform the request against the named
// DNS-server.
func (s *DNSTest) localQuery(server string, qname string, lookupType string) (*dns.Msg, error) {

	// Here we have a map of DNS type-names.
	var StringToType = map[string]uint16{
		"A":    dns.TypeA,
		"AAAA": dns.TypeAAAA,
		"MX":   dns.TypeMX,
		"NS":   dns.TypeNS,
		"TXT":  dns.TypeTXT,
	}

	qtype := StringToType[lookupType]
	if qtype == 0 {
		return nil, fmt.Errorf("unsupported record to lookup '%s'", lookupType)
	}
	localm.SetQuestion(qname, qtype)

	//
	// Default to connecting to an IPv4-address
	//
	address := fmt.Sprintf("%s:%d", server, 53)

	//
	// If we find a ":" we know it is an IPv6 address though
	//
	if strings.Contains(server, ":") {
		address = fmt.Sprintf("[%s]:%d", server, 53)
	}

	//
	// Run the lookup
	//
	r, _, err := localc.Exchange(localm, address)
	if err != nil {
		return nil, err
	}
	if r == nil || r.Rcode == dns.RcodeNameError || r.Rcode == dns.RcodeSuccess {
		return r, err
	}
	return nil, nil
}

// Arguments returns the names of arguments which this protocol-test
// understands, along with corresponding regular-expressions to validate
// their values.
func (s *DNSTest) Arguments() map[string]string {

	known := map[string]string{
		"type":   "A|AAAA|MX|NS|TXT",
		"lookup": ".*",
		"result": ".*",
	}
	return known
}

func (s *DNSTest) ShouldResolveHostname() bool {
	return true
}

// Example returns sample usage-instructions for self-documentation purposes.
func (s *DNSTest) Example() string {
	str := `
DNS Tester
----------
 The DNS tester allows you to confirm that the specified DNS server
 returns the results you expect.  It is invoked with input like this:

    ns.example.com must run dns with lookup test.example.com with type A with result '1.2.3.4'

 This test ensures that the DNS lookup of an A record for 'test.example.com'
 returns the single value 1.2.3.4

 Lookups are supported for A, AAAA, MX, NS, and TXT records.  If you expect
 there to be zero returning records, perhaps because you're ensuring that a
 service is IPv4-only you can specify that you require an empty result:

    rache.ns.cloudflare.com must run dns with lookup alert.steve.fi with type AAAA with result ''
`
	return str
}

// RunTest is the part of our API which is invoked to actually execute a
// test against the given target.
//
// In this case we make a DNS-lookup against the named host, and compare
// the result with what the user specified.
// look for a response which appears to be an FTP-server.
func (s *DNSTest) RunTest(tst test.Test, target string, opts test.Options) error {

	if tst.Arguments["lookup"] == "" {
		return errors.New("no value to lookup specified")
	}
	if tst.Arguments["type"] == "" {
		return errors.New("no record-type to lookup")
	}

	//
	// NOTE:
	// "result" must also be specified, but it is valid to set that
	// to be empty.
	//

	//
	// Run the lookup
	//
	res, err := s.lookup(target, tst.Arguments["lookup"], tst.Arguments["type"], opts.Timeout)
	if err != nil {
		return err
	}

	//
	// If the results differ that's an error
	//
	// Sort the results and comma-join for comparison
	//
	sort.Strings(res)
	found := strings.Join(res, ",")

	if found != tst.Arguments["result"] {
		return fmt.Errorf("expected DNS result to be '%s', but found '%s'", tst.Arguments["result"], found)
	}

	return nil

}

// Register our protocol-tester.
func init() {
	Register("dns", func() ProtocolTest {
		return &DNSTest{}
	})
}
