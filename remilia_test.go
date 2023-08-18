package remilia

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	name := "Test"
	r := New(name)

	if r.Name != name {
		t.Errorf("Expected name: %v, got: %v", name, r.Name)
	}
}

func TestWithOptions(t *testing.T) {
	r := New("Test")
	r2 := r.WithOptions(Delay(time.Second), AllowedDomains("test.com"))

	if r == r2 {
		t.Errorf("Expected r and r2 to be different instances")
	}

	if r2.Delay != time.Second {
		t.Errorf("Expected delay: %v, got %v", time.Second, r2.Delay)
	}

	if len(r2.AllowedDomains) != 1 || r2.AllowedDomains[0] != "test.com" {
		t.Errorf("Expected AllowedDomains: ['test.com'], got: %v", r2.AllowedDomains)
	}
}

func TestClone(t *testing.T) {
	r := New("Test")
	r2 := r.clone()

	if r == r2 {
		t.Errorf("Expected r and r2 to be different instances")
	}

	if r.Name != r2.Name || r.Delay != r2.Delay {
		t.Errorf("Expected r and r2 to have the same fields")
	}
}

func TestAllowedDomains(t *testing.T) {
	r := New("Test").WithOptions(AllowedDomains("test.com"))

	if len(r.AllowedDomains) != 1 || r.AllowedDomains[0] != "test.com" {
		t.Errorf("Expected AllowedDomains: ['test.com'], got %v", r.AllowedDomains)
	}
}

func TestDelay(t *testing.T) {
	r := New("Test").WithOptions(Delay(time.Second))

	if r.Delay != time.Second {
		t.Errorf("Expected Delay: %v, got: %v", time.Second, r.Delay)
	}
}
