package telemetry

import (
	"testing"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func notification(path string, val string) *gnmipb.Notification {
	return &gnmipb.Notification{
		Timestamp: 1700000000000000000,
		Update: []*gnmipb.Update{
			{
				Path: &gnmipb.Path{Elem: []*gnmipb.PathElem{{Name: path}}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: val}},
			},
		},
	}
}

func TestCacheUpdateAndGet(t *testing.T) {
	c := NewCache()
	c.Update("pe1", notification("oper-status", "UP"))

	v, ok := c.Get("pe1", "/oper-status")
	if !ok {
		t.Fatal("expected value to be present")
	}
	if v.Data != "UP" {
		t.Errorf("got %v, want UP", v.Data)
	}

	if _, ok := c.Get("pe1", "/nonexistent"); ok {
		t.Errorf("expected nonexistent path to be absent")
	}
	if _, ok := c.Get("nonexistent-device", "/oper-status"); ok {
		t.Errorf("expected nonexistent device to be absent")
	}
}

func TestCacheOverwritesOnUpdate(t *testing.T) {
	c := NewCache()
	c.Update("pe1", notification("oper-status", "DOWN"))
	c.Update("pe1", notification("oper-status", "UP"))

	v, ok := c.Get("pe1", "/oper-status")
	if !ok || v.Data != "UP" {
		t.Errorf("expected latest value UP, got %v (ok=%v)", v.Data, ok)
	}
}

func TestCacheDelete(t *testing.T) {
	c := NewCache()
	c.Update("pe1", notification("oper-status", "UP"))

	del := &gnmipb.Notification{
		Delete: []*gnmipb.Path{{Elem: []*gnmipb.PathElem{{Name: "oper-status"}}}},
	}
	c.Update("pe1", del)

	if _, ok := c.Get("pe1", "/oper-status"); ok {
		t.Errorf("expected path to be deleted")
	}
}

func TestCacheSnapshotAndDevices(t *testing.T) {
	c := NewCache()
	c.Update("pe1", notification("oper-status", "UP"))
	c.Update("pe2", notification("oper-status", "DOWN"))

	devs := c.Devices()
	if len(devs) != 2 {
		t.Fatalf("got %d devices, want 2", len(devs))
	}

	snap := c.Snapshot("pe1")
	if len(snap) != 1 || snap["/oper-status"].Data != "UP" {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}
