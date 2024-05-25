package rfc2136

const (
	// Name of this Extension
	Name = "rfc2136"

	// Type is the type of resources managed by the PowerDNS actuator.
	Type = "rfc2136"
)

// Credentials derrived from the Extensions Secret
// referenced in DNSRecord.spec.secretRef
type Credentials struct {
	// defaults to dynamic resolving SOA record from Zone, specified in DNSRecord.spec.zone
	Server      *string
	TsigKeyName string
	TsigSecret  string
	// defaults to "hmac-sha256.""
	Alogrithm string
}
