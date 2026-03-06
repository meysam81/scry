package crawler

import (
	"testing"
)

func TestFrontier_AddAndDequeue(t *testing.T) {
	f := NewFrontier("example.com", 100, nil, nil)

	if !f.Add("http://example.com/a", 0) {
		t.Error("expected Add to return true for first URL")
	}
	if !f.Add("http://example.com/b", 1) {
		t.Error("expected Add to return true for second URL")
	}

	task, ok := f.Dequeue()
	if !ok {
		t.Fatal("expected Dequeue to return true")
	}
	if task.URL != "http://example.com/a" {
		t.Errorf("URL = %q, want %q", task.URL, "http://example.com/a")
	}
	if task.Depth != 0 {
		t.Errorf("Depth = %d, want 0", task.Depth)
	}

	task, ok = f.Dequeue()
	if !ok {
		t.Fatal("expected Dequeue to return true")
	}
	if task.URL != "http://example.com/b" {
		t.Errorf("URL = %q, want %q", task.URL, "http://example.com/b")
	}

	_, ok = f.Dequeue()
	if ok {
		t.Error("expected Dequeue to return false on empty queue")
	}
}

func TestFrontier_Dedup(t *testing.T) {
	f := NewFrontier("example.com", 100, nil, nil)

	if !f.Add("http://example.com/page", 0) {
		t.Error("first Add should succeed")
	}
	if f.Add("http://example.com/page", 0) {
		t.Error("duplicate Add should return false")
	}
	if f.Seen() != 1 {
		t.Errorf("Seen = %d, want 1", f.Seen())
	}
}

func TestFrontier_HostScope(t *testing.T) {
	f := NewFrontier("example.com", 100, nil, nil)

	if f.Add("http://other.com/page", 0) {
		t.Error("should reject URL from different host")
	}
	if f.Seen() != 0 {
		t.Errorf("Seen = %d, want 0", f.Seen())
	}
}

func TestFrontier_MaxPages(t *testing.T) {
	f := NewFrontier("example.com", 2, nil, nil)

	if !f.Add("http://example.com/1", 0) {
		t.Error("first Add should succeed")
	}
	if !f.Add("http://example.com/2", 0) {
		t.Error("second Add should succeed")
	}
	if f.Add("http://example.com/3", 0) {
		t.Error("third Add should fail (maxPages=2)")
	}
}

func TestFrontier_IncludePatterns(t *testing.T) {
	f := NewFrontier("example.com", 100, []string{"/blog/*"}, nil)

	if !f.Add("http://example.com/blog/post1", 0) {
		t.Error("should match /blog/* pattern")
	}
	if f.Add("http://example.com/about", 0) {
		t.Error("should reject URL not matching include pattern")
	}
}

func TestFrontier_ExcludePatterns(t *testing.T) {
	f := NewFrontier("example.com", 100, nil, []string{"/admin/*"})

	if !f.Add("http://example.com/page", 0) {
		t.Error("should allow URL not matching exclude pattern")
	}
	if f.Add("http://example.com/admin/settings", 0) {
		t.Error("should reject URL matching exclude pattern")
	}
}

func TestFrontier_LenAndSeen(t *testing.T) {
	f := NewFrontier("example.com", 100, nil, nil)

	f.Add("http://example.com/a", 0)
	f.Add("http://example.com/b", 0)
	f.Add("http://example.com/c", 0)

	if f.Len() != 3 {
		t.Errorf("Len = %d, want 3", f.Len())
	}
	if f.Seen() != 3 {
		t.Errorf("Seen = %d, want 3", f.Seen())
	}

	f.Dequeue()

	if f.Len() != 2 {
		t.Errorf("Len = %d, want 2", f.Len())
	}
	if f.Seen() != 3 {
		t.Errorf("Seen = %d, want 3 (Seen should not decrease)", f.Seen())
	}
}
