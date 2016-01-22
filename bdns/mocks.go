package bdns

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/letsencrypt/boulder/Godeps/_workspace/src/github.com/miekg/dns"
	"github.com/letsencrypt/boulder/Godeps/_workspace/src/golang.org/x/net/context"
)

// MockDNSResolver is a mock
type MockDNSResolver struct {
}

// LookupTXT is a mock
func (mock *MockDNSResolver) LookupTXT(_ context.Context, hostname string) ([]string, []string, error) {
	if hostname == "_acme-challenge.servfail.com" {
		return nil, nil, fmt.Errorf("SERVFAIL")
	}
	if hostname == "_acme-challenge.good-dns01.com" {
		// base64(sha256("LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0"
		//               + "." + "9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI"))
		// expected token + test account jwk thumbprint
		return []string{"LPsIwTo7o8BoG0-vjCyGQGBWSVIPxI-i_X336eUOQZo"}, []string{"respect my authority!"}, nil
	}
	if hostname == "_acme-challenge.no-authority-dns01.com" {
		// base64(sha256("LoqXcYV8q5ONbJQxbmR7SCTNo3tiAXDfowyjxAjEuX0"
		//               + "." + "9jg46WB3rR_AHD-EBXdN7cBkH1WOu0tA3M9fm21mqTI"))
		// expected token + test account jwk thumbprint
		return []string{"LPsIwTo7o8BoG0-vjCyGQGBWSVIPxI-i_X336eUOQZo"}, nil, nil
	}
	return []string{"hostname"}, []string{"respect my authority!"}, nil
}

// MockTimeoutError returns a a net.OpError for which Timeout() returns true.
func MockTimeoutError() *net.OpError {
	return &net.OpError{
		Err: os.NewSyscallError("ugh timeout", timeoutError{}),
	}
}

type timeoutError struct{}

func (t timeoutError) Error() string {
	return "so sloooow"
}
func (t timeoutError) Timeout() bool {
	return true
}

// LookupHost is a mock
//
// Note: see comments on LookupMX regarding email.only
//
func (mock *MockDNSResolver) LookupHost(_ context.Context, hostname string) ([]net.IP, error) {
	if hostname == "always.invalid" ||
		hostname == "invalid.invalid" ||
		hostname == "email.only" {
		return []net.IP{}, nil
	}
	if hostname == "always.timeout" {
		return []net.IP{}, &dnsError{dns.TypeA, "always.timeout", MockTimeoutError(), -1}
	}
	if hostname == "always.error" {
		return []net.IP{}, &dnsError{dns.TypeA, "always.error", &net.OpError{
			Err: errors.New("some net error"),
		}, -1}
	}
	ip := net.ParseIP("127.0.0.1")
	return []net.IP{ip}, nil
}

// LookupCAA is a mock
func (mock *MockDNSResolver) LookupCAA(_ context.Context, domain string) ([]*dns.CAA, error) {
	var results []*dns.CAA
	var record dns.CAA
	switch strings.TrimRight(domain, ".") {
	case "caa-timeout.com":
		return nil, &dnsError{dns.TypeCAA, "always.timeout", MockTimeoutError(), -1}
	case "reserved.com":
		record.Tag = "issue"
		record.Value = "symantec.com"
		results = append(results, &record)
	case "critical.com":
		record.Flag = 1
		record.Tag = "issue"
		record.Value = "symantec.com"
		results = append(results, &record)
	case "present.com":
		record.Tag = "issue"
		record.Value = "letsencrypt.org"
		results = append(results, &record)
	case "com":
		// Nothing should ever call this, since CAA checking should stop when it
		// reaches a public suffix.
		fallthrough
	case "servfail.com":
		return results, fmt.Errorf("SERVFAIL")
	}
	return results, nil
}

// LookupMX is a mock
//
// Note: the email.only domain must have an MX but no A or AAAA
// records. The mock LookupHost returns an address of 127.0.0.1 for
// all domains except for special cases, so MX-only domains must be
// handled in both LookupHost and LookupMX.
//
func (mock *MockDNSResolver) LookupMX(_ context.Context, domain string) ([]string, error) {
	switch strings.TrimRight(domain, ".") {
	case "letsencrypt.org":
		fallthrough
	case "email.only":
		fallthrough
	case "email.com":
		return []string{"mail.email.com"}, nil
	}
	return nil, nil
}
