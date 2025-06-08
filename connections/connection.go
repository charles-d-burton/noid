package connections

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/tailscale/tailscale-client-go/tailscale"
	"tailscale.com/tsnet"
	"tailscale.com/types/logger"
)

// Enum type for Auth Type
type AuthType int

// Enum definition for Auth Type
const (
	OAUTH AuthType = iota
	AUTHKEY
	NONE
)

// Tailnet main struct to hold connection to the tailnet information
type Tailnet struct {
	ConfigDir    string
	ClientID     string
	ClientSecret string
	AuthKey      string
	Hostname     string
	Addr         string
	Port         string
	Scopes       []string
	Tags         []string
	Client       *tailscale.Client
	TSServer     *tsnet.Server
	Listener     net.Listener
	authType     AuthType
	logger       logger.Logf
}

// Option function to set different options on the tailnet config
type Option func(tn *Tailnet) error

// connect to the tailnet using oauth credentials
func (tn *Tailnet) Connect(ctx context.Context, opts ...Option) error {
	for _, opt := range opts {
		err := opt(tn)
		if err != nil {
			return err
		}
	}

	if tn.Hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			return err
		}
		tn.Hostname = h + "-tailsys"
	}

	err := tn.initClient(ctx)
	if err != nil {
		return err
	}
	// var logger *slog.Logger
	// logger.
	// if !tn.TailnetLogging {
	//   logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	// } else {
	//   logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	// }

	srv := &tsnet.Server{
		Hostname:  tn.Hostname,
		AuthKey:   tn.AuthKey,
		Ephemeral: true,
		Logf:      tn.logger, // func(string, ...any) {},
	}
	tn.TSServer = srv
	tn.authType = tn.getAuthType()

	return nil
}

// ConnectCmd setup the tailnet connection without starting the gRPC server
func (tn *Tailnet) ConnectCmd(ctx context.Context, opts ...Option) error {
	for _, opt := range opts {
		err := opt(tn)
		if err != nil {
			return err
		}
	}

	if tn.Hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			return err
		}
		tn.Hostname = h + "-tailsys"
	}

	err := tn.initClient(ctx)
	if err != nil {
		return err
	}

	srv := &tsnet.Server{
		Hostname:  tn.Hostname,
		AuthKey:   tn.AuthKey,
		Ephemeral: true,
		Logf:      tn.logger,
	}
	tn.TSServer = srv
	tn.authType = tn.getAuthType()

	return nil
}

func (tn *Tailnet) initClient(ctx context.Context) error {
	var capabilities tailscale.KeyCapabilities
	capabilities.Devices.Create.Reusable = true
	capabilities.Devices.Create.Ephemeral = true
	capabilities.Devices.Create.Tags = tn.Tags
	capabilities.Devices.Create.Preauthorized = true

	var topts []tailscale.CreateKeyOption
	topts = append(topts, tailscale.WithKeyExpiry(10*time.Second))

	if tn.authType == OAUTH {
		slog.Debug("connecting with oauth")
		client, err := tailscale.NewClient(
			"",
			"-",
			tailscale.WithOAuthClientCredentials(tn.ClientID, tn.ClientSecret, tn.Scopes),
		)
		if err != nil {
			return err
		}
		key, err := client.CreateKey(ctx, capabilities, topts...)
		if err != nil {
			return err
		}
		tn.AuthKey = key.Key
		tn.Client = client
		tn.reapDeviceID(ctx)
		return nil
	} else if tn.authType == AUTHKEY {
		slog.Debug("connecting with authkey")
		client, err := tailscale.NewClient(tn.AuthKey, "-")
		if err != nil {
			return err
		}
		tn.Client = client
		tn.reapDeviceID(ctx)
	}
	return nil
}

func (tn *Tailnet) reapDeviceID(ctx context.Context) error {
	devices, err := tn.Client.Devices(ctx)
	slog.Debug(fmt.Sprintf("FOUND %d DEVICES", len(devices)))
	if err != nil {
		return err
	}
	for _, device := range devices {
		slog.Debug(device.Hostname)
		if device.Hostname == tn.Hostname {
			slog.Debug(fmt.Sprintf("found device %s with same name %s\n\n", device.Hostname, tn.Hostname))
			err := tn.Client.DeleteDevice(ctx, device.ID)
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (tn *Tailnet) checkForKeys() bool {
	// tc := TLSConfig{}
	// c, err := os.ReadFile(tn.ConfigDir + "/certs/certs.yaml")
	// if err != nil {
	// 	return false
	// }
	// err = yaml.Unmarshal(c, &tc)
	// if err != nil {
	// 	return false
	// }
	// tn.TLSConfig = &tc
	return true
}

// GetDevices returns a list of devices that are connected to the configured tailnet
func (tn *Tailnet) GetDevices(ctx context.Context) ([]tailscale.Device, error) {
	return tn.Client.Devices(ctx)
}

// WithOauth sets up the tailnet connection using an oauth credential
func (tn *Tailnet) WithOauth(clientId, clientSecret string) Option {
	return func(tn *Tailnet) error {
		if clientId == "" {
			return errors.New("client id not set")
		}

		if clientSecret == "" {
			return errors.New("client secret not set")
		}
		tn.ClientID = clientId
		tn.ClientSecret = clientSecret
		return nil
	}
}

// WithAPIKey sets the Option to connect to the tailnet with a preconfigured Auth key
func (tn *Tailnet) WithAuthKey(key string) Option {
	return func(tn *Tailnet) error {
		tn.AuthKey = key
		return nil
	}
}

// WithScopes sets the Oauth scopes to configure for the connection
func (tn *Tailnet) WithScopes(scopes ...string) Option {
	return func(tn *Tailnet) error {
		if scopes != nil {
			tn.Scopes = scopes
		}
		return nil
	}
}

// WithTags sets the tags that were configured with the oauth connection
func (tn *Tailnet) WithTags(tags ...string) Option {
	return func(tn *Tailnet) error {
		if tags != nil {
			for _, tag := range tags {
				stag := strings.Split(tag, ":")
				if len(stag) < 2 {
					return errors.New(fmt.Sprintf("tag %s mush be in format tag:<tag>", tag))
				}
			}
			tn.Tags = tags
		}
		return nil
	}
}

// WithHostname Override the hostname on the tailnet
func (tn *Tailnet) WithHostname(hostname string) Option {
	return func(tn *Tailnet) error {
		if hostname == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}
			tn.Hostname = hostname + "-tailsys"
			return nil
		}
		tn.Hostname = hostname
		return nil
	}
}

// WithTailnetLogging Enable/Disable logging on the tailnet
func (tn *Tailnet) WithTailnetLogging(enabled bool) Option {
	return func(tn *Tailnet) error {
		if !enabled {
			tn.logger = logger.Discard
			return nil
		}
		return nil
	}
}

// WithPort Port to bind the grpc server to
func (tn *Tailnet) WithPort(port string) Option {
	return func(tn *Tailnet) error {
		tn.Port = port
		return nil
	}
}

func (tn *Tailnet) WithConfigDir(dir string) Option {
	return func(tn *Tailnet) error {
		tn.ConfigDir = dir
		slog.Debug(fmt.Sprintf("set config dir to: %s", tn.ConfigDir))
		return nil
	}
}

// getAuthType Determine the type of auth to connect to the tailnet
func (tn *Tailnet) getAuthType() AuthType {
	if tn.ClientID != "" && tn.ClientSecret != "" {
		slog.Debug("auth type is OAUTH")
		return OAUTH
	}

	if tn.AuthKey != "" {
		slog.Debug("auth type is AUTHKEY")
		return AUTHKEY

	}
	slog.Debug("auth type is NONE")
	return NONE
}
