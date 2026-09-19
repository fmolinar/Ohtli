// Package telemetry holds the normalized, device-independent state cache
// fed by gNMI Subscribe updates (README.md section 7's "State cache" box).
package telemetry

import (
	"sync"
	"time"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/fmolinar/Ohtli/controller/internal/gnmi"
)

// Value is a single cached leaf: its current value and when it was last
// updated.
type Value struct {
	Data      interface{}
	Timestamp time.Time
}

// Cache stores the latest known value of every subscribed path, per
// device. Safe for concurrent use by multiple gNMI subscription
// goroutines and readers.
type Cache struct {
	mu    sync.RWMutex
	state map[string]map[string]Value // device -> path string -> value
}

func NewCache() *Cache {
	return &Cache{state: make(map[string]map[string]Value)}
}

// Update applies every update (and processes every delete) in a gNMI
// Notification for the given device.
func (c *Cache) Update(device string, notif *gnmipb.Notification) {
	if notif == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	dev, ok := c.state[device]
	if !ok {
		dev = make(map[string]Value)
		c.state[device] = dev
	}

	ts := time.Unix(0, notif.Timestamp)
	prefix := gnmi.PathToString(notif.Prefix)
	if prefix == "/" {
		prefix = ""
	}

	for _, upd := range notif.Update {
		path := prefix + gnmi.PathToString(upd.Path)
		dev[path] = Value{
			Data:      gnmi.ValueToInterface(upd.Val),
			Timestamp: ts,
		}
	}
	for _, del := range notif.Delete {
		path := prefix + gnmi.PathToString(del)
		delete(dev, path)
	}
}

// Get returns the current value of a single path for a device.
func (c *Cache) Get(device, path string) (Value, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dev, ok := c.state[device]
	if !ok {
		return Value{}, false
	}
	v, ok := dev[path]
	return v, ok
}

// Snapshot returns a copy of every cached value for a device.
func (c *Cache) Snapshot(device string) map[string]Value {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dev := c.state[device]
	out := make(map[string]Value, len(dev))
	for k, v := range dev {
		out[k] = v
	}
	return out
}

// Devices returns the names of every device with at least one cached
// value.
func (c *Cache) Devices() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.state))
	for name := range c.state {
		names = append(names, name)
	}
	return names
}
