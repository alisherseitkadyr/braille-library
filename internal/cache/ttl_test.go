package cache

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	now := time.Now()
	c := NewTTL[string, int](time.Minute)
	c.now = func() time.Time { return now }

	if _, ok := c.Get("a"); ok {
		t.Fatal("empty cache returned a value")
	}
	c.Set("a", 1)
	c.Set("b", 2)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a) = %v, %v; want 1, true", v, ok)
	}

	c.Delete("a")
	if _, ok := c.Get("a"); ok {
		t.Fatal("deleted key is still cached")
	}

	now = now.Add(59 * time.Second)
	if _, ok := c.Get("b"); !ok {
		t.Fatal("entry expired too early")
	}
	now = now.Add(time.Second)
	if _, ok := c.Get("b"); ok {
		t.Fatal("entry did not expire after the TTL")
	}
	if len(c.entries) != 0 {
		t.Fatalf("expired entry was not removed: %v", c.entries)
	}

	c.Set("x", 1)
	c.Set("y", 2)
	c.Clear()
	if _, ok := c.Get("x"); ok {
		t.Fatal("Clear left entries")
	}
}
