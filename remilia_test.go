package remilia

import (
	"testing"
)

func TestNew(t *testing.T) {
	url := "www.test.com"
	r := New(url)

	if r.URL != url {
		t.Errorf("Expected url: %v, got: %v", url, r.URL)
	}
}

func TestWithOptions(t *testing.T) {
	AppName := "Remilia"

	r := New("www.test.com")
	r2 := r.withOptions(Name(AppName))

	if r == r2 {
		t.Errorf("Expected r and r2 to be different instances")
	}

	if r2.Name != "Remilia" {
		t.Errorf("Expected name: %v, got %v", AppName, r2.Name)
	}
}

func TestClone(t *testing.T) {
	r := New("www.test.com")
	r2 := r.clone()

	if r == r2 {
		t.Errorf("Expected r and r2 to be different instances")
	}

	if r.Name != r2.Name || r.Delay != r2.Delay {
		t.Errorf("Expected r and r2 to have the same fields")
	}
}

func TestConcurrentNumber(t *testing.T) {
	r := New("www.test.com").withOptions(ConcurrentNumber(20))

	if r.ConcurrentNumber != 20 {
		t.Errorf("Expected ConcurrentNumber: 20, got %v", r.ConcurrentNumber)
	}
}

func TestName(t *testing.T) {
	r := New("www.test.com").withOptions(Name("test"))

	if r.Name != "test" {
		t.Errorf("Expected Name: 'test', got %v", r.Name)
	}
}
