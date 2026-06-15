package classcentral

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient builds a Client pointed at srv with rate-limiting disabled.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0 // no pacing in tests
	cfg.Retries = 2
	cfg.Timeout = 5 * time.Second
	return NewClient(cfg)
}

// ─── fixture helpers ──────────────────────────────────────────────────────────

func searchFixture(courses ...wireCourse) []byte {
	b, _ := json.Marshal(searchResp{Courses: courses, TotalCount: len(courses)})
	return b
}

func subjectsFixture(subjects ...wireSubject) []byte {
	b, _ := json.Marshal(subjectsResp{Subjects: subjects})
	return b
}

func providersFixture(providers ...wireProvider) []byte {
	b, _ := json.Marshal(providersResp{Providers: providers})
	return b
}

func blockPage() []byte {
	return []byte(`<!DOCTYPE html><html><head><title>Just a moment...</title></head><body><p>cf-browser-verification</p></body></html>`)
}

func blockPageLower() []byte {
	return []byte(`<!doctype html><html><head><title>Please wait...</title></head><body></body></html>`)
}

// ─── Search tests ─────────────────────────────────────────────────────────────

func TestSearchReturnsCourses(t *testing.T) {
	fixture := searchFixture(wireCourse{
		ID:              1,
		Name:            "Machine Learning",
		Slug:            "machine-learning-835",
		Rating:          4.9,
		ReviewCount:     180000,
		EnrollmentCount: 5000000,
		Level:           "Beginner",
		Language:        "English",
		Certificate:     true,
		DurationWeeks:   11,
		HoursPerWeek:    5,
		Providers:       []wireProvider{{Name: "Coursera"}},
		Universities:    []wireUniv{{Name: "Stanford University"}},
		URL:             "https://www.classcentral.com/course/machine-learning-835",
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "machine learning", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(courses) != 1 {
		t.Fatalf("want 1 course, got %d", len(courses))
	}
	cr := courses[0]
	if cr.Name != "Machine Learning" {
		t.Errorf("Name = %q", cr.Name)
	}
	if cr.Provider != "Coursera" {
		t.Errorf("Provider = %q", cr.Provider)
	}
	if cr.Institution != "Stanford University" {
		t.Errorf("Institution = %q", cr.Institution)
	}
	if cr.Rating != 4.9 {
		t.Errorf("Rating = %v", cr.Rating)
	}
	if cr.Rank != 1 {
		t.Errorf("Rank = %d, want 1", cr.Rank)
	}
}

func TestSearchRankAssignment(t *testing.T) {
	fixture := searchFixture(
		wireCourse{Name: "Course A", Slug: "course-a"},
		wireCourse{Name: "Course B", Slug: "course-b"},
		wireCourse{Name: "Course C", Slug: "course-c"},
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "test", false, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for i, cr := range courses {
		if cr.Rank != i+1 {
			t.Errorf("courses[%d].Rank = %d, want %d", i, cr.Rank, i+1)
		}
	}
}

func TestSearchFreeFilter(t *testing.T) {
	var gotFree string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFree = r.URL.Query().Get("free")
		_, _ = w.Write(searchFixture())
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Search(context.Background(), "python", true, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotFree != "true" {
		t.Errorf("free param = %q, want %q", gotFree, "true")
	}
}

// ─── Top tests ────────────────────────────────────────────────────────────────

func TestTopCourses(t *testing.T) {
	var gotSort, gotFree string
	fixture := searchFixture(
		wireCourse{Name: "Python for Everybody", Slug: "python-1"},
		wireCourse{Name: "Machine Learning", Slug: "ml-835"},
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSort = r.URL.Query().Get("sort")
		gotFree = r.URL.Query().Get("free")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Top(context.Background(), 10)
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	if gotSort != "student-count" {
		t.Errorf("sort param = %q, want %q", gotSort, "student-count")
	}
	if gotFree != "true" {
		t.Errorf("free param = %q, want %q", gotFree, "true")
	}
	if len(courses) != 2 {
		t.Errorf("want 2 courses, got %d", len(courses))
	}
}

// ─── Subjects tests ───────────────────────────────────────────────────────────

func TestSubjects(t *testing.T) {
	fixture := subjectsFixture(
		wireSubject{Name: "Computer Science", Slug: "cs", CourseCount: 5432},
		wireSubject{Name: "Data Science", Slug: "data-science", CourseCount: 3210},
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	subjects, err := c.Subjects(context.Background())
	if err != nil {
		t.Fatalf("Subjects: %v", err)
	}
	if len(subjects) != 2 {
		t.Fatalf("want 2 subjects, got %d", len(subjects))
	}
	s := subjects[0]
	if s.Name != "Computer Science" {
		t.Errorf("Name = %q", s.Name)
	}
	if s.Count != 5432 {
		t.Errorf("Count = %d", s.Count)
	}
	if s.Rank != 1 {
		t.Errorf("Rank = %d", s.Rank)
	}
	if s.URL == "" {
		t.Error("URL is empty")
	}
}

// ─── Providers tests ──────────────────────────────────────────────────────────

func TestProviders(t *testing.T) {
	fixture := providersFixture(
		wireProvider{Name: "Coursera", Slug: "coursera", CourseCount: 8765},
		wireProvider{Name: "edX", Slug: "edx", CourseCount: 4321},
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	providers, err := c.Providers(context.Background())
	if err != nil {
		t.Fatalf("Providers: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("want 2 providers, got %d", len(providers))
	}
	p := providers[0]
	if p.Name != "Coursera" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.Courses != 8765 {
		t.Errorf("Courses = %d", p.Courses)
	}
	if p.URL == "" {
		t.Error("URL is empty")
	}
}

// ─── Cloudflare / blocking tests ──────────────────────────────────────────────

func TestBlockedHTMLResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(blockPage())
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Search(context.Background(), "python", false, 5)
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("want ErrBlocked, got %v", err)
	}
}

func TestBlockedHTMLResponseLowercase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(blockPageLower())
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Subjects(context.Background())
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("want ErrBlocked, got %v", err)
	}
}

// ─── Retry tests ──────────────────────────────────────────────────────────────

func TestRetriesOn503(t *testing.T) {
	var hits int
	fixture := searchFixture(wireCourse{Name: "Recovered", Slug: "recovered"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	cfg.Timeout = 5 * time.Second
	c := NewClient(cfg)

	start := time.Now()
	courses, err := c.Search(context.Background(), "test", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
	if len(courses) == 0 || courses[0].Name != "Recovered" {
		t.Errorf("unexpected courses: %v", courses)
	}
}

func TestNoRetryOn403(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	cfg.Timeout = 5 * time.Second
	c := NewClient(cfg)

	_, err := c.Search(context.Background(), "test", false, 5)
	if err == nil {
		t.Fatal("expected error on 403")
	}
	if hits != 1 {
		t.Errorf("server saw %d hits, want 1 (no retry on 403)", hits)
	}
}

// ─── Field mapping tests ──────────────────────────────────────────────────────

func TestCourseRatingZero(t *testing.T) {
	fixture := searchFixture(wireCourse{Name: "No Ratings Yet", Slug: "no-ratings", Rating: 0.0})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "test", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if courses[0].Rating != 0.0 {
		t.Errorf("Rating = %v, want 0.0", courses[0].Rating)
	}
}

func TestDurationFormatWeeksAndHours(t *testing.T) {
	fixture := searchFixture(wireCourse{
		Name:          "Deep Learning",
		Slug:          "deep-learning",
		DurationWeeks: 11,
		HoursPerWeek:  5,
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "deep learning", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := "11 weeks, 5 hrs/week"
	if courses[0].Duration != want {
		t.Errorf("Duration = %q, want %q", courses[0].Duration, want)
	}
}

func TestDurationFormatWeeksOnly(t *testing.T) {
	fixture := searchFixture(wireCourse{
		Name:          "Short Course",
		Slug:          "short-course",
		DurationWeeks: 4,
		HoursPerWeek:  0,
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "short", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := "4 weeks"
	if courses[0].Duration != want {
		t.Errorf("Duration = %q, want %q", courses[0].Duration, want)
	}
}

func TestCertificateTrue(t *testing.T) {
	fixture := searchFixture(wireCourse{Name: "Cert Course", Slug: "cert-course", Certificate: true})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "cert", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !courses[0].Certificate {
		t.Error("Certificate should be true")
	}
}

func TestSlugURL(t *testing.T) {
	// URL field absent; slug present -- should derive URL from slug.
	fixture := searchFixture(wireCourse{Name: "Slug Course", Slug: "foo-bar-123", URL: ""})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "slug", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := "https://www.classcentral.com/course/foo-bar-123"
	if courses[0].URL != want {
		t.Errorf("URL = %q, want %q", courses[0].URL, want)
	}
}

func TestInstitutionExtracted(t *testing.T) {
	fixture := searchFixture(wireCourse{
		Name: "CS101", Slug: "cs101",
		Universities: []wireUniv{
			{Name: "MIT"},
			{Name: "Harvard"},
		},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "cs", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	// Only the first university is used.
	if courses[0].Institution != "MIT" {
		t.Errorf("Institution = %q, want MIT", courses[0].Institution)
	}
}

func TestProviderExtracted(t *testing.T) {
	fixture := searchFixture(wireCourse{
		Name: "Data 101", Slug: "data-101",
		Providers: []wireProvider{
			{Name: "Coursera"},
			{Name: "edX"},
		},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	courses, err := c.Search(context.Background(), "data", false, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	// Only the first provider is used.
	if courses[0].Provider != "Coursera" {
		t.Errorf("Provider = %q, want Coursera", courses[0].Provider)
	}
}
