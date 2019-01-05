package bpa

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	URISchemeDTN uint = 1
	URISchemeIPN uint = 2
)

// EndpointID represents an Endpoint ID as defined in section 4.1.5.1. The
// "scheme name" is represented by an uint (vide supra) and the "scheme-specific
// part" (SSP) by an interface{}. Based on the characteristic of the name, the
// underlying type of the SSP may vary.
type EndpointID struct {
	_struct struct{} `codec:",toarray"`

	SchemeName         uint
	SchemeSpecificPort interface{}
}

func newEndpointIDDTN(ssp string) (EndpointID, error) {
	var sspRaw interface{}
	if ssp == "none" {
		sspRaw = uint(0)
	} else {
		sspRaw = string(ssp)
	}

	return EndpointID{
		SchemeName:         URISchemeDTN,
		SchemeSpecificPort: sspRaw,
	}, nil
}

func newEndpointIDIPN(ssp string) (ep EndpointID, err error) {
	// As definied in RFC 6260, section 2.1:
	// - node number: ASCII numeric digits between 1 and (2^64-1)
	// - an ASCII dot
	// - service number: ASCII numeric digits between 1 and (2^64-1)

	re := regexp.MustCompile(`^(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(ssp)
	if len(matches) != 3 {
		err = newBPAError("IPN does not satisfy given regex")
		return
	}

	nodeNo, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return
	}

	serviceNo, err := strconv.ParseUint(matches[2], 10, 64)
	if err != nil {
		return
	}

	if nodeNo < 1 || serviceNo < 1 {
		err = newBPAError("IPN's node and service number must be >= 1")
		return
	}

	ep = EndpointID{
		SchemeName:         URISchemeIPN,
		SchemeSpecificPort: [2]uint64{nodeNo, serviceNo},
	}
	return
}

// NewEndpointID creates a new EndpointID by a given "scheme name" and a
// "scheme-specific part" (SSP). Currently the "dtn" and "ipn"-scheme names
// are supported.
func NewEndpointID(name, ssp string) (EndpointID, error) {
	switch name {
	case "dtn":
		return newEndpointIDDTN(ssp)
	case "ipn":
		return newEndpointIDIPN(ssp)
	default:
		return EndpointID{}, newBPAError("Unknown scheme type")
	}
}

// setEndpointIDFromCborArray sets the fields of the EndpointID addressed by
// the EndpointID-pointer based on the given array. This function is used for
// the CBOR decoding of the PrimaryBlock and some Extension Blocks.
func setEndpointIDFromCborArray(ep *EndpointID, arr []interface{}) {
	(*ep).SchemeName = uint(arr[0].(uint64))
	(*ep).SchemeSpecificPort = arr[1]

	// The codec library uses uint64 for uints and []interface{} for arrays
	// internally. However, our `dtn:none` is defined by an uint and each "ipn"
	// endpoint by an uint64 array. That's why we have to re-cast some types..

	switch ty := reflect.TypeOf((*ep).SchemeSpecificPort); ty.Kind() {
	case reflect.Uint64:
		(*ep).SchemeSpecificPort = uint((*ep).SchemeSpecificPort.(uint64))

	case reflect.Slice:
		(*ep).SchemeSpecificPort = [2]uint64{
			(*ep).SchemeSpecificPort.([]interface{})[0].(uint64),
			(*ep).SchemeSpecificPort.([]interface{})[1].(uint64),
		}
	}
}

func (eid EndpointID) String() string {
	var b strings.Builder

	switch eid.SchemeName {
	case URISchemeDTN:
		b.WriteString("dtn")
	case URISchemeIPN:
		b.WriteString("ipn")
	default:
		fmt.Fprintf(&b, "unknown_%d", eid.SchemeName)
	}
	b.WriteRune(':')

	switch t := eid.SchemeSpecificPort.(type) {
	case uint:
		if eid.SchemeName == URISchemeDTN && eid.SchemeSpecificPort.(uint) == 0 {
			b.WriteString("none")
		} else {
			fmt.Fprintf(&b, "%d", eid.SchemeSpecificPort.(uint))
		}

	case string:
		b.WriteString(eid.SchemeSpecificPort.(string))

	case [2]uint64:
		var ssp [2]uint64 = eid.SchemeSpecificPort.([2]uint64)
		if eid.SchemeName == URISchemeIPN {
			fmt.Fprintf(&b, "%d.%d", ssp[0], ssp[1])
		} else {
			fmt.Fprintf(&b, "%v", ssp)
		}

	default:
		fmt.Fprintf(&b, "unkown %T: %v", t, eid.SchemeSpecificPort)
	}

	return b.String()
}

// DtnNone returns the "dtn:none" endpoint.
func DtnNone() EndpointID {
	return EndpointID{
		SchemeName:         URISchemeDTN,
		SchemeSpecificPort: uint(0),
	}
}
