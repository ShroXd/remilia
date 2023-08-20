package remilia

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	url := "www.test.com"
	r := New(url)

	if r.URL != url {
		t.Errorf("Expected url: %v, got: %v", url, r.URL)
	}
}

func TestWithOptions(t *testing.T) {
	r := New("www.test.com")
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
	r := New("www.test.com").WithOptions(ConcurrentNumber(20))

	if r.ConcurrentNumber != 20 {
		t.Errorf("Expected ConcurrentNumber: 20, got %v", r.ConcurrentNumber)
	}
}

func TestName(t *testing.T) {
	r := New("www.test.com").WithOptions(Name("test"))

	if r.Name != "test" {
		t.Errorf("Expected Name: 'test', got %v", r.Name)
	}
}

func TestAllowedDomains(t *testing.T) {
	r := New("www.test.com").WithOptions(AllowedDomains("test.com"))

	if len(r.AllowedDomains) != 1 || r.AllowedDomains[0] != "test.com" {
		t.Errorf("Expected AllowedDomains: ['test.com'], got %v", r.AllowedDomains)
	}
}

func TestDelay(t *testing.T) {
	r := New("www.test.com").WithOptions(Delay(time.Second))

	if r.Delay != time.Second {
		t.Errorf("Expected Delay: %v, got: %v", time.Second, r.Delay)
	}
}

func TestUserAgent(t *testing.T) {
	fakeUserAgent := "Mozilla/5.0 (FakeOS; TestingEnvironment; NOT-REAL-BROWSER) Gecko/20100101 FakeBrowser/0.0.1 FOR-TESTING-PURPOSES-ONLY"
	r := New("Test").WithOptions(UserAgent(fakeUserAgent))

	if r.UserAgent != fakeUserAgent {
		t.Errorf("Expected UserAgent: %v, got: %v", fakeUserAgent, r.UserAgent)
	}
}
