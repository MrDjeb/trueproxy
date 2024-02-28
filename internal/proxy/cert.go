package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"time"

	"github.com/mrdjeb/trueproxy/internal/config"
)

type CertManager struct {
	ca           *x509.Certificate // Root certificate
	caPrivateKey *rsa.PrivateKey   // CA private key

	roots *x509.CertPool

	privateKey *rsa.PrivateKey

	validity     time.Duration
	keyID        []byte
	organization string
}

func NewCertManager(cfg config.Cert) (*CertManager, error) {

	tlsCert, err := tls.LoadX509KeyPair(cfg.CACertFile, cfg.CAKeyFile)
	if err != nil {
		return nil, err
	}

	privateKey, ok := tlsCert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("failed cast caPrivateKey")
	}

	ca, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, err
	}

	roots := x509.NewCertPool()
	roots.AddCert(ca)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pub := priv.Public()

	pkixpub, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	h := sha1.New()
	_, err = h.Write(pkixpub)
	if err != nil {
		return nil, err
	}
	keyID := h.Sum(nil)

	return &CertManager{
		ca:           ca,
		caPrivateKey: privateKey,
		privateKey:   priv,
		keyID:        keyID,
		validity:     time.Hour,
		organization: cfg.Organization,
		roots:        roots,
	}, nil
}

func (c *CertManager) GenFakeCert(hostname string) (*tls.Certificate, error) {

	host, _, err := net.SplitHostPort(hostname)
	if err == nil {
		hostname = host
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   hostname,
			Organization: []string{c.organization},
		},
		SubjectKeyId:          c.keyID,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		NotBefore:             time.Now().Add(-c.validity),
		NotAfter:              time.Now().Add(c.validity),
	}

	if ip := net.ParseIP(hostname); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{hostname}
	}

	raw, err := x509.CreateCertificate(rand.Reader, tmpl, c.ca, c.privateKey.Public(), c.caPrivateKey)
	if err != nil {
		return nil, err
	}

	x509c, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{raw, c.ca.Raw},
		PrivateKey:  c.privateKey,
		Leaf:        x509c,
	}

	return cert, nil

}

func (c *CertManager) NewTLSConfig(hostname string) *tls.Config {
	tlsConfig := &tls.Config{
		GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			host := clientHello.ServerName
			if host == "" {
				host = hostname
			}
			return c.GenFakeCert(host)
		},
		NextProtos: []string{"http/1.1"},
	}
	tlsConfig.InsecureSkipVerify = true
	return tlsConfig
}
