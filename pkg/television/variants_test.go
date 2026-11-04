package television

import "testing"

func names(chs []Channel) []string {
	out := make([]string, len(chs))
	for i, c := range chs {
		out[i] = c.Name
	}
	return out
}

func TestPreferLanguageVariants(t *testing.T) {
	in := []Channel{
		{Name: "Discovery English"},
		{Name: "Discovery Hindi"},
		{Name: "Discovery Tamil"},
		{Name: "sonic Tamil"},
		{Name: "Sonic Hindi"},
		{Name: "Colors HD"},
		{Name: "Colors Tamil HD"},
		{Name: "News18 Marathi"},
		{Name: "News18 Kannada"},
		{Name: "Aaj Tak"},
		{Name: "History TV18 HD"},
		{Name: "History TV18 HD Hindi"},
	}
	got := names(PreferLanguageVariants(in, "Hindi"))
	want := []string{"Discovery Hindi", "Sonic Hindi", "Colors HD", "Colors Tamil HD", "News18 Marathi", "News18 Kannada", "Aaj Tak", "History TV18 HD Hindi"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPreferLanguageVariantsOff(t *testing.T) {
	in := []Channel{{Name: "Discovery English"}, {Name: "Discovery Hindi"}}
	for _, v := range []string{"", "off", "OFF", "none"} {
		if got := PreferLanguageVariants(in, v); len(got) != 2 {
			t.Fatalf("%q: got %v", v, names(got))
		}
	}
	if got := PreferLanguageVariants(in, "tamil"); len(got) != 2 {
		t.Fatalf("no tamil variant, nothing should drop: %v", names(got))
	}
}
