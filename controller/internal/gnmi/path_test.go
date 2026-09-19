package gnmi

import (
	"testing"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestParsePathAndPathToStringRoundTrip(t *testing.T) {
	cases := []string{
		"/interfaces/interface[name=eth1]/state/oper-status",
		"/network-instances/network-instance[name=default]/protocols",
		"/components/component/state",
		"/",
		"",
	}

	for _, in := range cases {
		p, err := ParsePath(in)
		if err != nil {
			t.Fatalf("ParsePath(%q): unexpected error: %v", in, err)
		}

		want := in
		if want == "" {
			want = "/"
		}
		if got := PathToString(p); got != want {
			t.Errorf("PathToString(ParsePath(%q)) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePathRejectsMalformed(t *testing.T) {
	cases := []string{
		"/interfaces/interface[name",
		"/interfaces/interface[noequals]",
		"//",
	}
	for _, in := range cases {
		if _, err := ParsePath(in); err == nil {
			t.Errorf("ParsePath(%q): expected error, got nil", in)
		}
	}
}

func TestValueToInterface(t *testing.T) {
	cases := []struct {
		name string
		tv   *gnmipb.TypedValue
		want interface{}
	}{
		{"string", &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "up"}}, "up"},
		{"int", &gnmipb.TypedValue{Value: &gnmipb.TypedValue_IntVal{IntVal: -5}}, int64(-5)},
		{"uint", &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 5}}, uint64(5)},
		{"bool", &gnmipb.TypedValue{Value: &gnmipb.TypedValue_BoolVal{BoolVal: true}}, true},
		{"nil", nil, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ValueToInterface(c.tv)
			if got != c.want {
				t.Errorf("ValueToInterface(%v) = %v (%T), want %v (%T)", c.tv, got, got, c.want, c.want)
			}
		})
	}
}
