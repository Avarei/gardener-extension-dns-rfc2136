package client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

// Client contains configuration and provides functions that use said configuration for the creation of RFC2136 records.
type Client struct {
	Server        *string
	TsigKeyName   string
	TsigSecret    string
	TsigAlgorithm string
}

// NewClient creates a new Client
func NewClient(tsigKeyName, tsigSecret, tsigAlgorithm string, server *string) *Client {
	return &Client{
		Server:        server,
		TsigKeyName:   tsigKeyName,
		TsigSecret:    tsigSecret,
		TsigAlgorithm: tsigAlgorithm,
	}
}

// CreateOrUpdate an A, AAAA, CNAME, or TXT record
func (c *Client) CreateOrUpdate(ctx context.Context, zone, name string, recordType string, value []string, ttl int64) error {
	zone = dns.Fqdn(zone)
	name = dns.Fqdn(name)

	rr := createRRs(name, dns.StringToType[string(recordType)], value, uint32(ttl))

	msg := new(dns.Msg)
	msg.SetUpdate(zone)
	msg.Insert(rr)
	msg.SetTsig(c.TsigKeyName, c.TsigAlgorithm, 300, time.Now().Unix())
	client := new(dns.Client)
	client.TsigSecret = map[string]string{c.TsigKeyName: c.TsigSecret}
	server, err := c.getDNSServer(ctx, zone)
	if err != nil {
		return err
	}
	resp, _, err := client.Exchange(msg, server)
	if err != nil {
		return err
	}

	if resp.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("failure during %s", dns.RcodeToString[resp.Rcode])
	}
	return nil

}

func (c *Client) getDNSServer(ctx context.Context, zone string) (string, error) {
	if c.Server != nil {
		return ensurePortOnServer(*c.Server), nil
	}
	soa, err := getSOARecord(ctx, zone)
	if err != nil {
		return "", err
	}
	authServer := soa.Ns
	return ensurePortOnServer(authServer), nil
}

// GetZone attempts to read the SOA Record of fqdn reducing shortening the domain
// by one lever with each unsuccessfull iteration
func GetZone(ctx context.Context, fqdn string) (string, error) {
	fqdn = dns.Fqdn(fqdn)

	labels := strings.Split(fqdn, ".")[:1]
	for i := 0; i < len(labels); i++ {
		parentFqdn := strings.Join(labels[i:], ".") + "."
		soa, err := getSOARecord(ctx, parentFqdn)
		if err == nil {
			return soa.Ns, nil
		}
	}
	return "", fmt.Errorf("could not find zone for %s", fqdn)
}

func getSOARecord(ctx context.Context, zone string) (*dns.SOA, error) {
	zone = dns.Fqdn(zone)

	m := new(dns.Msg)
	m.SetQuestion(zone, dns.TypeSOA)
	m.RecursionDesired = true
	c := new(dns.Client)
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, errors.Wrap(err, "error evaluating /etc/resolv.conf")
	}

	server := config.Servers[0] + ":" + config.Port
	r, _, err := c.ExchangeContext(ctx, m, server)
	if err != nil {
		fmt.Printf("DNS query failed: %v\n", err)
		return nil, fmt.Errorf("SOA query for %s at %s failed", zone, server)
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("SOA record for %s could not be found by %s", zone, server)
	}

	for _, rr := range r.Answer {
		if soa, ok := rr.(*dns.SOA); ok {
			return soa, nil
		}
	}
	return nil, fmt.Errorf("unable to find soa record for zone %s at %s", zone, server)
}

func createRRs(name string, recordType uint16, value []string, ttl uint32) []dns.RR {
	name = dns.Fqdn(name)
	var rr []dns.RR

	switch recordType {
	case dns.TypeA:
		for _, content := range value {
			rr = append(rr, &dns.A{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    ttl,
				},
				A: net.ParseIP(content),
			})
		}
	case dns.TypeAAAA:
		for _, content := range value {
			rr = append(rr, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    ttl,
				},
				AAAA: net.ParseIP(content),
			})
		}
	case dns.TypeCNAME:
		rr = append(rr, &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Target: value[0],
		})
	case dns.TypeTXT:
		rr = append(rr, &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Txt: value,
		})
	default:
		return nil
	}
	return rr
}

// Delete removes the given DNS Record
func (c *Client) Delete(ctx context.Context, zone, name string, recordType string, value []string) error {
	zone = dns.Fqdn(zone)
	name = dns.Fqdn(name)

	rr := createRRs(name, dns.StringToType[string(recordType)], value, 0)

	server, err := c.getDNSServer(ctx, zone)
	if err != nil {
		return err
	}

	// Create a new DNS message for the update request
	m := new(dns.Msg)
	m.SetUpdate(zone)

	// Add the delete instruction to the message
	m.Remove(rr)

	// Create a DNS client and send the update request
	client := new(dns.Client)
	resp, _, err := client.Exchange(m, server)
	if err != nil {
		return errors.Wrapf(err, "error deleteing %s record %s", recordType, name)
	}

	// Check the response for success
	if resp.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("Failed to delete DNS record: %s", dns.RcodeToString[resp.Rcode])
	}

	return nil

}

func ensurePortOnServer(server string) string {
	if !strings.Contains(server, ":") {
		return server + ":53"
	}
	return server
}
