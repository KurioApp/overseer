// IMAPS Tester
//
// The IMAPS tester connects to a remote host and ensures that this
// succeeds.  If you supply a username & password a login will be
// made, and the test will fail if this login fails.
//
// This test is invoked via input like so:
//
//    host.example.com must run imap [with username 'steve@steve' with password 'secret']
//
// Because IMAPS uses TLS it will test the validity of the certificate as
// part of the test, if you wish to disable this add `with tls insecure`.
//

package protocols

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"

	client "github.com/emersion/go-imap/client"
	"github.com/skx/overseer/test"
)

// IMAPSTest is our object
type IMAPSTest struct {
}

// Arguments returns the names of arguments which this protocol-test
// understands, along with corresponding regular-expressions to validate
// their values.
func (s *IMAPSTest) Arguments() map[string]string {
	known := map[string]string{
		"port":     "^[0-9]+$",
		"tls":      "insecure",
		"username": ".*",
		"password": ".*",
	}
	return known
}

func (s *IMAPSTest) ShouldResolveHostname() bool {
	return true
}

// Example returns sample usage-instructions for self-documentation purposes.
func (s *IMAPSTest) Example() string {
	str := `
IMAPS Tester
------------
 The IMAPS tester connects to a remote host and ensures that this succeeds.

 If you supply a username & password a login will be made, and the test will
 fail if this login does not succeed.

 This test is invoked via input like so:

    host.example.com must run imaps

 Because IMAPS uses TLS this test will ensure the validity of the certificate as
 part of the test, if you wish to disable this add "with tls insecure".
`

	return str
}

// RunTest is the part of our API which is invoked to actually execute a
// test against the given target.
//
// In this case we make a IMAP connection to the specified host, and if
// a username + password were specified we then attempt to authenticate
// to the remote host too.
func (s *IMAPSTest) RunTest(tst test.Test, target string, opts test.Options) error {
	var err error

	//
	// The default port to connect to.
	//
	port := 993

	//
	// If the user specified a different port update to use it.
	//
	if tst.Arguments["port"] != "" {
		port, err = strconv.Atoi(tst.Arguments["port"])
		if err != nil {
			return err
		}
	}

	//
	// Should we skip validation of the SSL certificate?
	//
	insecure := false
	if tst.Arguments["tls"] == "insecure" {
		insecure = true
	}

	//
	// Default to connecting to an IPv4-address
	//
	address := fmt.Sprintf("%s:%d", target, port)

	//
	// If we find a ":" we know it is an IPv6 address though
	//
	if strings.Contains(target, ":") {
		address = fmt.Sprintf("[%s]:%d", target, port)
	}

	//
	// Setup a dialer so we can have a suitable timeout
	//
	var dial = &net.Dialer{
		Timeout: opts.Timeout,
	}

	//
	// Setup the default TLS config.
	//
	// We need to setup the hostname that the TLS certificate
	// will verify upon, from our input-line.
	//
	data := strings.Fields(tst.Input)
	tlsSetup := &tls.Config{ServerName: data[0]}

	//
	// Disable verification if we're being insecure.
	//
	if insecure {
		tlsSetup = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	//
	// Connect.
	//
	con, err := client.DialWithDialerTLS(dial, address, tlsSetup)
	if err != nil {
		return err

	}
	defer con.Close()

	//
	// If we got username/password then use them
	//
	if (tst.Arguments["username"] != "") && (tst.Arguments["password"] != "") {
		err = con.Login(tst.Arguments["username"], tst.Arguments["password"])
		if err != nil {
			return err
		}

		// Logout so that we don't keep the handle open.
		err = con.Logout()
		if err != nil {
			return err
		}
	}

	return nil
}

//
// Register our protocol-tester.
//
func init() {
	Register("imaps", func() ProtocolTest {
		return &IMAPSTest{}
	})
}
