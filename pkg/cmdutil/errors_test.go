package cmdutil

import (
	"errors"
	"testing"
)

func TestErrorEnvelopeCarriesTheFix(t *testing.T) {
	err := WithFix(errors.New("no roam root"), "orgx ws add main --root ~/org/roam")
	env := NewErrorEnvelope(err)
	if env.Kind != "orgx.error.v1" || env.Error.Message != "no roam root" || env.Error.Fix == "" {
		t.Fatalf("envelope = %+v", env)
	}
	plain := NewErrorEnvelope(errors.New("boom"))
	if plain.Error.Message != "boom" || plain.Error.Fix != "" {
		t.Fatalf("plain envelope = %+v", plain)
	}
}
