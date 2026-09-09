package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/control"
)

type fakePasskeys struct {
	keys map[uuid.UUID][]control.UserPasskey
	err  error
}

func (f fakePasskeys) ListByUserID(_ context.Context, id uuid.UUID) ([]control.UserPasskey, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.keys[id], nil
}

// A HALF-WIRED PROVIDER IS NOT A WAY IN.
//
// A client id with no secret is somebody who started configuring a provider and
// did not finish. Counting it would wave through the exact lockout this guard
// exists to prevent — and it is the likeliest shape for the mistake to take,
// since the id is the half that is not secret and therefore the half that gets
// pasted in first.
func TestAProviderWithNoSecretIsNotAnAlternative(t *testing.T) {
	c := ConfiguredSignInAlternatives{
		Providers: []SocialProviderConfig{
			{Name: "Google", ClientID: "abc.apps.googleusercontent.com", ClientSecret: ""},
			{Name: "GitHub", ClientID: "", ClientSecret: "shhh"},
		},
	}
	ok, how, err := c.Available(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("a half-configured provider counted as a way in (%q)", how)
	}
}

func TestAFullyConfiguredProviderIsAnAlternative(t *testing.T) {
	c := ConfiguredSignInAlternatives{
		Providers: []SocialProviderConfig{
			{Name: "Keycloak", ClientID: "id", ClientSecret: "secret"},
		},
	}
	ok, how, err := c.Available(context.Background(), uuid.Nil)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v, want a configured provider to count", ok, err)
	}
	if how == "" {
		t.Fatal("the alternative was not named, so a refusal elsewhere cannot say what to do")
	}
}

// The acting user's own passkey is a weaker guarantee than a provider — it only
// gets ONE person back in — but that is the difference between an inconvenience
// and an outage, because the person it gets back in is the one who made the
// change.
func TestTheActingUsersPasskeyCountsAsAWayBack(t *testing.T) {
	me := uuid.New()
	c := ConfiguredSignInAlternatives{
		Passkeys: fakePasskeys{keys: map[uuid.UUID][]control.UserPasskey{me: {{}}}},
	}

	ok, _, err := c.Available(context.Background(), me)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v, want the acting user's passkey to count", ok, err)
	}

	// Somebody else's passkey does not help the person making the change.
	ok, _, err = c.Available(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("a passkey belonging to another user counted; it does not get the operator " +
			"making this change back in")
	}
}

// REPORTED, NOT SWALLOWED. On the refusal path of a write, an unreadable store
// must not read as "no alternative" — that refuses a legitimate change on a
// blip — nor as "an alternative exists", which waves a lockout through.
func TestAnUnreadablePasskeyStoreIsAnErrorNotAnAnswer(t *testing.T) {
	boom := errors.New("database is down")
	c := ConfiguredSignInAlternatives{Passkeys: fakePasskeys{err: boom}}

	ok, _, err := c.Available(context.Background(), uuid.New())
	if err == nil {
		t.Fatalf("an unreadable passkey store answered %v instead of failing", ok)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the underlying failure preserved", err)
	}
	if ok {
		t.Fatal("it answered true as well as erroring, which a careless caller would act on")
	}
}
