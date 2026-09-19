package gnmi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// TLSConfig configures how a Client authenticates to a target. README.md
// section 14 requires mTLS for gNMI; Insecure exists only for early lab
// bring-up against a target that doesn't have certificates issued yet and
// must never be set for anything reachable outside the lab.
type TLSConfig struct {
	Insecure           bool
	InsecureSkipVerify bool
	CAFile             string
	CertFile           string
	KeyFile            string
}

// Client wraps a gNMI gRPC client for a single target.
type Client struct {
	Target string

	conn *grpc.ClientConn
	gc   gnmipb.GNMIClient
}

// Dial connects to a gNMI target at addr ("host:port").
func Dial(ctx context.Context, addr string, tlsCfg TLSConfig) (*Client, error) {
	creds, err := transportCredentials(tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("gnmi: building transport credentials for %s: %w", addr, err)
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gnmi: dialing %s: %w", addr, err)
	}

	return &Client{
		Target: addr,
		conn:   conn,
		gc:     gnmipb.NewGNMIClient(conn),
	}, nil
}

// NewFromClient wraps an existing gnmipb.GNMIClient (e.g. a bufconn client
// in tests) rather than dialing a real address.
func NewFromClient(target string, gc gnmipb.GNMIClient) *Client {
	return &Client{Target: target, gc: gc}
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func transportCredentials(cfg TLSConfig) (credentials.TransportCredentials, error) {
	if cfg.Insecure {
		return insecure.NewCredentials(), nil
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // explicit opt-in, documented as lab-only
	}

	if cfg.CAFile != "" {
		pool := x509.NewCertPool()
		pem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("reading CA file %s: %w", cfg.CAFile, err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificates found in CA file %s", cfg.CAFile)
		}
		tlsConf.RootCAs = pool
	}

	if cfg.CertFile != "" || cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading client cert/key: %w", err)
		}
		tlsConf.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsConf), nil
}

// Capabilities calls the gNMI Capabilities RPC.
func (c *Client) Capabilities(ctx context.Context) (*gnmipb.CapabilityResponse, error) {
	resp, err := c.gc.Capabilities(ctx, &gnmipb.CapabilityRequest{})
	if err != nil {
		return nil, fmt.Errorf("gnmi: Capabilities against %s: %w", c.Target, err)
	}
	return resp, nil
}

// Get calls the gNMI Get RPC for the given paths.
func (c *Client) Get(ctx context.Context, paths []string) (*gnmipb.GetResponse, error) {
	req := &gnmipb.GetRequest{
		Type: gnmipb.GetRequest_ALL,
	}
	for _, p := range paths {
		parsed, err := ParsePath(p)
		if err != nil {
			return nil, err
		}
		req.Path = append(req.Path, parsed)
	}

	resp, err := c.gc.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gnmi: Get against %s: %w", c.Target, err)
	}
	return resp, nil
}

// Subscribe opens a STREAM subscription for the given paths and invokes
// onUpdate for every Notification received (sync_response messages are
// skipped). It blocks until ctx is cancelled or the stream ends.
func (c *Client) Subscribe(ctx context.Context, paths []string, onUpdate func(*gnmipb.Notification)) error {
	subs := make([]*gnmipb.Subscription, 0, len(paths))
	for _, p := range paths {
		parsed, err := ParsePath(p)
		if err != nil {
			return err
		}
		subs = append(subs, &gnmipb.Subscription{
			Path: parsed,
			Mode: gnmipb.SubscriptionMode_TARGET_DEFINED,
		})
	}

	stream, err := c.gc.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("gnmi: opening Subscribe stream to %s: %w", c.Target, err)
	}

	req := &gnmipb.SubscribeRequest{
		Request: &gnmipb.SubscribeRequest_Subscribe{
			Subscribe: &gnmipb.SubscriptionList{
				Subscription: subs,
				Mode:         gnmipb.SubscriptionList_STREAM,
			},
		},
	}
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("gnmi: sending SubscribeRequest to %s: %w", c.Target, err)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("gnmi: Subscribe stream from %s ended: %w", c.Target, err)
		}
		switch r := resp.Response.(type) {
		case *gnmipb.SubscribeResponse_Update:
			onUpdate(r.Update)
		case *gnmipb.SubscribeResponse_SyncResponse:
			// Initial sync complete; nothing to do for a read-only cache.
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}
