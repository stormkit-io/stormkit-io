package integrations

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"golang.org/x/crypto/ssh"
)

type HetznerInterface interface {
	CreateServer(ctx context.Context, opts CreateServerOpts) (*ServerOutput, error)
}

type hclient struct {
	client *hcloud.Client
}

func Hetzner() HetznerInterface {
	return &hclient{
		client: hcloud.NewClient(hcloud.WithToken(os.Getenv("HETZNER_API_KEY"))),
	}
}

type CreateServerOpts struct {
	UserID types.ID
}

type DataCenter struct {
	ID       int64
	Name     string
	Location string
}

type ServerOutput struct {
	ServerID   int64
	IPv4       net.IP
	IPv6       net.IP
	SSHKey     *SSHKey
	DataCenter *DataCenter
}

func (h *hclient) createPlacementGroup(ctx context.Context, name string) (int64, error) {
	pgName := fmt.Sprintf("pg-%s", name)
	result, _, err := h.client.PlacementGroup.Get(ctx, pgName)

	if err != nil {
		return 0, err
	}

	if result != nil && result.ID != 0 {
		return result.ID, nil
	}

	insertResult, _, err := h.client.PlacementGroup.Create(ctx, hcloud.PlacementGroupCreateOpts{
		Name: pgName,
		Type: hcloud.PlacementGroupTypeSpread,
	})

	if err != nil {
		return 0, err
	}

	if insertResult.PlacementGroup == nil {
		return 0, errors.New("placement group is empty")
	}

	return insertResult.PlacementGroup.ID, nil
}

type SSHKey struct {
	KeyName    string
	KeyID      int64
	PrivateKey string
}

func (h *hclient) createSSHKey(ctx context.Context, clientID types.ID) (*SSHKey, error) {
	keyName := fmt.Sprintf("ssh-%s", clientID.String())

	// Try getting first
	sshKey, _, err := h.client.SSHKey.Get(ctx, keyName)
	result := &SSHKey{}

	if err != nil {
		return nil, err
	}

	if sshKey == nil {
		privateKey, publicKey, err := generateSSHKeyInMemory(2048)

		if err != nil {
			return nil, err
		}

		sshKey, _, err = h.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
			Labels: map[string]string{
				"client-id": clientID.String(),
			},
			Name:      keyName,
			PublicKey: publicKey,
		})

		if err != nil {
			return nil, err
		}

		if privateKey != "" {
			result.PrivateKey = privateKey
		}
	}

	if sshKey == nil {
		return nil, errors.New("ssh key not generated")
	}

	result.KeyID = sshKey.ID
	result.KeyName = sshKey.Name

	return result, nil
}

// CreateServer creates a new server in Hetzner Cloud.
func (h *hclient) CreateServer(ctx context.Context, opts CreateServerOpts) (*ServerOutput, error) {
	keyResult, err := h.createSSHKey(ctx, opts.UserID)

	if err != nil {
		return nil, err
	}

	pgID, err := h.createPlacementGroup(ctx, opts.UserID.String())

	if err != nil {
		return nil, err
	}

	userData, err := yaml.Marshal(map[string]any{
		"runcmd": []string{
			"sudo apt-get update -y",
			"sudo apt-get install -y ca-certificates curl",
			"sudo install -m 0755 -d /etc/apt/keyrings",
			"sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc",
			"sudo chmod a+r /etc/apt/keyrings/docker.asc",
			`echo \
			  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
			  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
			  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null`,
			"sudo apt-get update -y",
			"sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin",
		},
	})

	if err != nil {
		return nil, err
	}

	result, response, err := h.client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:           fmt.Sprintf("server-%s-%s", opts.UserID.String(), uuid.New().String()),
		Image:          &hcloud.Image{Name: "ubuntu-24.04"},
		Datacenter:     &hcloud.Datacenter{Name: "nbg1-dc3"},
		Firewalls:      []*hcloud.ServerCreateFirewall{{Firewall: hcloud.Firewall{ID: 1619062}}},
		PlacementGroup: &hcloud.PlacementGroup{ID: pgID},
		Labels: map[string]string{
			"client-id": opts.UserID.String(),
		},
		PublicNet: &hcloud.ServerCreatePublicNet{
			EnableIPv4: true,
			EnableIPv6: true,
			IPv4:       nil,
			IPv6:       nil,
		},
		ServerType: &hcloud.ServerType{Name: "cx22"},
		SSHKeys: []*hcloud.SSHKey{
			{ID: keyResult.KeyID},
			{ID: 23077199}, // personal ssh key for debugging
		},
		StartAfterCreate: hcloud.Ptr(true),
		UserData:         "#cloud-config\n" + string(userData),
	})

	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, errors.New("hcloud server create response is nil")
	}

	output := &ServerOutput{
		SSHKey: keyResult,
	}

	if result.Server != nil {
		output.ServerID = result.Server.ID
		output.IPv4 = result.Server.PublicNet.IPv4.IP
		output.IPv6 = result.Server.PublicNet.IPv6.IP
		output.DataCenter = &DataCenter{
			Name:     result.Server.Datacenter.Name,
			ID:       result.Server.Datacenter.ID,
			Location: fmt.Sprintf("%s:%s", result.Server.Datacenter.Location.Country, result.Server.Datacenter.Location.City),
		}
	}

	return output, nil
}

func generateSSHKeyInMemory(bits int) (privateKey string, publicKey string, err error) {
	// Generate private key
	rsaKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", err
	}

	// Convert private key to PEM format
	var privKeyBuf bytes.Buffer

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	}

	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", err
	}

	// Generate public key
	publicRsaKey, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	return privKeyBuf.String(), string(ssh.MarshalAuthorizedKey(publicRsaKey)), nil
}
