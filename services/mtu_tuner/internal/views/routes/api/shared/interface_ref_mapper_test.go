package shared

import (
	"testing"

	"mtu-tuner/internal/core"
	apitypes "mtu-tuner/internal/views/routes/api/_gen_types"
)

func TestInterfaceRefCoreMapsStableIdentityFields(t *testing.T) {
	t.Parallel()

	ref := &apitypes.InterfaceRef{
		PlatformName: "Windows",
		Name:         "Wi-Fi",
		Index:        "12",
	}

	info := InterfaceRefCore(ref)

	want := core.InterfaceInfo{
		PlatformName: "Windows",
		Name:         "Wi-Fi",
		Index:        "12",
	}
	if info != want {
		t.Fatalf("InterfaceRefCore() = %#v, want %#v", info, want)
	}
}

func TestInterfaceRefCoreReturnsZeroValueForNilRef(t *testing.T) {
	t.Parallel()

	if info := InterfaceRefCore(nil); info != (core.InterfaceInfo{}) {
		t.Fatalf("InterfaceRefCore(nil) = %#v, want zero value", info)
	}
}
