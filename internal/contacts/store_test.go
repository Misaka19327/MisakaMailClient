package contacts

import (
	"testing"
)

func TestMergeDedupesAndPrefersName(t *testing.T) {
	existing := []Contact{
		{Name: "", Email: "a@x.com", Source: "sent"},
		{Name: "Bob", Email: "b@x.com", Source: "sent"},
	}
	fresh := []Contact{
		{Name: "Alice", Email: "a@x.com", Source: "inbox"}, // fills name, source stays lower? a is sent(2) vs inbox(1) -> keep sent
		{Name: "Bob New", Email: "b@x.com", Source: "vcard"}, // name already set (Bob), source vcard(3) > sent(2) -> vcard
		{Name: "Carol", Email: "c@x.com", Source: "sent"},
	}
	merged := Merge(existing, fresh)
	if len(merged) != 3 {
		t.Fatalf("got %d contacts, want 3: %+v", len(merged), merged)
	}
	byEmail := map[string]Contact{}
	for _, c := range merged {
		byEmail[c.Email] = c
	}
	if byEmail["a@x.com"].Name != "Alice" {
		t.Errorf("a name: got %q want Alice", byEmail["a@x.com"].Name)
	}
	if byEmail["a@x.com"].Source != "sent" {
		t.Errorf("a source: got %q want sent (higher rank)", byEmail["a@x.com"].Source)
	}
	if byEmail["b@x.com"].Name != "Bob" {
		t.Errorf("b name: got %q want Bob (existing kept)", byEmail["b@x.com"].Name)
	}
	if byEmail["b@x.com"].Source != "vcard" {
		t.Errorf("b source: got %q want vcard (higher rank)", byEmail["b@x.com"].Source)
	}
}

func TestMergeEmpty(t *testing.T) {
	if got := Merge(nil, nil); len(got) != 0 {
		t.Errorf("merge(nil,nil) = %v, want empty", got)
	}
}
