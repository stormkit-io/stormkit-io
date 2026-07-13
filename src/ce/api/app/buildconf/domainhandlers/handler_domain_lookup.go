package domainhandlers

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type certInfo struct {
	StartDate          utils.Unix `json:"startDate"`
	EndDate            utils.Unix `json:"endDate"`
	Subject            string     `json:"subject"`
	Issuer             issuerInfo `json:"issuer"`
	DNSNames           []string   `json:"dnsNames"`
	SerialNumber       string     `json:"serialNo"`
	SignatureAlgorithm string     `json:"signatureAlgorithm"`
}

type issuerInfo struct {
	Country      []string `json:"country"`
	Organization []string `json:"organization"`
	CommonName   string   `json:"name"`
}

type dnsInfo struct {
	// IsDomainVerified is set based on the txt record value.
	// If it matches the expected txt record it will be true, otherwise false.
	IsDomainVerified bool `json:"verified"`

	// TXT represents the txt record to set to verify the domain.
	TXT map[string]interface{} `json:"txt"`
}

type domainLookupResponse struct {
	TLSError error `json:"tlsError"`
	// TLS represents the response the certificate information.
	// It will be nil in cases where a domain is not in use.
	TLS *certInfo `json:"tls"`

	// DNS represents the dns information on the domain.
	DNS *dnsInfo `json:"dns"`

	DomainName string `json:"domainName"`
}

// handlerDomainLookup performs a dns lookup and verifies the domain if needed.
func handlerDomainLookup(req *app.RequestContext) *shttp.Response {
	store := buildconf.DomainStore()
	domain, err := store.DomainByID(req.Context(), utils.StringToID(req.Query().Get("domainId")))

	if err != nil {
		return shttp.Error(err)
	}

	if domain == nil {
		return shttp.NoContent()
	}

	dns := requestDNSRecords(domain.Name, req.App)

	if !domain.Verified && dns.IsDomainVerified {
		if err := store.VerifyDomain(req.Context(), domain.ID); err != nil {
			return shttp.Error(err)
		}
	}

	var tls *certInfo
	var tlsError error
	dns.IsDomainVerified = domain.Verified || dns.IsDomainVerified

	if dns.IsDomainVerified {
		tls, tlsError = requestTLSInfo(domain.Name)
	}

	return &shttp.Response{
		Data: &domainLookupResponse{
			TLSError:   tlsError,
			TLS:        tls,
			DNS:        dns,
			DomainName: domain.Name,
		},
	}
}

// requestDNSRecords will request the dns records that are needed
// to verify the domain ownership.
func requestDNSRecords(name string, a *app.App) *dnsInfo {
	txtName, txtValue := generateTXTRecord(a)
	txtLookupAddress := fmt.Sprintf("%s.%s", txtName, name)

	txtRecords, err := net.LookupTXT(txtLookupAddress)
	isDomainVerified := utils.InSliceStringCS(txtRecords, txtValue)

	return &dnsInfo{
		TXT: map[string]interface{}{
			"name":    txtName,
			"value":   txtValue,
			"lookup":  txtLookupAddress,
			"records": txtRecords,
			"err":     err,
		},
		IsDomainVerified: isDomainVerified,
	}
}

// requestTLSInfo returns information on the certificate.
func requestTLSInfo(domainName string) (*certInfo, error) {
	conf := tls.Config{}
	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: 10 & time.Second,
	}, "tcp", fmt.Sprintf("%s:443", domainName), &conf)

	if err != nil {
		return nil, err
	}

	defer conn.Close()

	certChain := conn.ConnectionState().PeerCertificates

	if len(certChain) == 0 {
		return nil, err
	}

	cert := certChain[0]

	return &certInfo{
		StartDate: utils.Unix{Time: cert.NotBefore, Valid: true},
		EndDate:   utils.Unix{Time: cert.NotAfter, Valid: true},
		Subject:   cert.Subject.CommonName,
		Issuer: issuerInfo{
			Country:      cert.Issuer.Country,
			Organization: cert.Issuer.Organization,
			CommonName:   cert.Issuer.CommonName,
		},
		DNSNames:           cert.DNSNames,
		SerialNumber:       cert.SerialNumber.String(),
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
	}, nil
}

// Based on the display name, generate a record value.
func generateTXTRecord(a *app.App) (name, value string) {
	b1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b1, uint64(a.ID))
	enc1 := md5.Sum(append(b1[:], 0x02))

	b2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b2, uint64(a.ID))
	enc2 := md5.Sum(append(b2[:], 0x03))

	return hex.EncodeToString(enc1[:]), hex.EncodeToString(enc2[:])
}
