package gnmi

import (
	"context"
	"net"
	"testing"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// fakeGNMIServer is a minimal in-memory gNMI target used to exercise
// Client against real gRPC plumbing without a live device.
type fakeGNMIServer struct {
	gnmipb.UnimplementedGNMIServer

	capabilitiesResp *gnmipb.CapabilityResponse
	getResp          *gnmipb.GetResponse
	notifications    []*gnmipb.Notification
}

func (f *fakeGNMIServer) Capabilities(ctx context.Context, req *gnmipb.CapabilityRequest) (*gnmipb.CapabilityResponse, error) {
	return f.capabilitiesResp, nil
}

func (f *fakeGNMIServer) Get(ctx context.Context, req *gnmipb.GetRequest) (*gnmipb.GetResponse, error) {
	return f.getResp, nil
}

func (f *fakeGNMIServer) Subscribe(stream gnmipb.GNMI_SubscribeServer) error {
	if _, err := stream.Recv(); err != nil {
		return err
	}
	for _, n := range f.notifications {
		if err := stream.Send(&gnmipb.SubscribeResponse{
			Response: &gnmipb.SubscribeResponse_Update{Update: n},
		}); err != nil {
			return err
		}
	}
	return stream.Send(&gnmipb.SubscribeResponse{
		Response: &gnmipb.SubscribeResponse_SyncResponse{SyncResponse: true},
	})
}

func dialFake(t *testing.T, srv *fakeGNMIServer) (*Client, func()) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	gnmipb.RegisterGNMIServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dialing bufconn: %v", err)
	}

	client := NewFromClient("fake", gnmipb.NewGNMIClient(conn))
	cleanup := func() {
		_ = conn.Close()
		gs.Stop()
	}
	return client, cleanup
}

func TestClientCapabilities(t *testing.T) {
	srv := &fakeGNMIServer{
		capabilitiesResp: &gnmipb.CapabilityResponse{
			SupportedModels: []*gnmipb.ModelData{{Name: "openconfig-interfaces"}},
		},
	}
	client, cleanup := dialFake(t, srv)
	defer cleanup()

	resp, err := client.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if len(resp.SupportedModels) != 1 || resp.SupportedModels[0].Name != "openconfig-interfaces" {
		t.Errorf("unexpected capabilities response: %+v", resp)
	}
}

func TestClientGet(t *testing.T) {
	srv := &fakeGNMIServer{
		getResp: &gnmipb.GetResponse{
			Notification: []*gnmipb.Notification{
				{Update: []*gnmipb.Update{{
					Path: &gnmipb.Path{Elem: []*gnmipb.PathElem{{Name: "oper-status"}}},
					Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "UP"}},
				}}},
			},
		},
	}
	client, cleanup := dialFake(t, srv)
	defer cleanup()

	resp, err := client.Get(context.Background(), []string{"/interfaces/interface[name=eth1]/state/oper-status"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(resp.Notification) != 1 {
		t.Fatalf("got %d notifications, want 1", len(resp.Notification))
	}
}

func TestClientSubscribe(t *testing.T) {
	want := &gnmipb.Notification{
		Update: []*gnmipb.Update{{
			Path: &gnmipb.Path{Elem: []*gnmipb.PathElem{{Name: "oper-status"}}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "UP"}},
		}},
	}
	srv := &fakeGNMIServer{notifications: []*gnmipb.Notification{want}}
	client, cleanup := dialFake(t, srv)
	defer cleanup()

	var got []*gnmipb.Notification
	ctx, cancel := context.WithCancel(context.Background())
	err := client.Subscribe(ctx, []string{"/interfaces/interface/state/oper-status"}, func(n *gnmipb.Notification) {
		got = append(got, n)
		if len(got) == len(srv.notifications) {
			cancel()
		}
	})
	if err != nil && ctx.Err() == nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d notifications, want 1", len(got))
	}
	if ValueToInterface(got[0].Update[0].Val) != "UP" {
		t.Errorf("unexpected update value: %+v", got[0])
	}
}
