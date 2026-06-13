package classcentral

import (
	"strings"
	"testing"
)

const baseURL = "https://www.classcentral.com"

var sampleNextDataPage = `<!DOCTYPE html>
<html>
<head></head>
<body>
<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"courses":[{"id":1,"name":"Python for Everybody","slug":"python-1","provider":{"name":"Coursera","slug":"coursera"},"institutions":[{"name":"University of Michigan"}],"rating":4.8,"reviewsCount":50000,"enrollmentsCount":2000000,"workload":"4-6 hours/week","certificate":true,"language":"English","level":"Beginner"}]}},"page":"/search","query":{}}</script>
</body>
</html>`

func TestNextData(t *testing.T) {
	nd, err := nextData([]byte(sampleNextDataPage))
	if err != nil {
		t.Fatalf("nextData error: %v", err)
	}
	if nd == nil {
		t.Fatal("nextData returned nil")
	}
	pp := pageProps(nd)
	if pp == nil {
		t.Fatal("pageProps returned nil")
	}
}

func TestExtractCoursesFromNextData(t *testing.T) {
	courses := parseCourses([]byte(sampleNextDataPage), baseURL)
	if len(courses) != 1 {
		t.Fatalf("want 1 course, got %d", len(courses))
	}
	c := courses[0]
	if c.Name != "Python for Everybody" {
		t.Errorf("name = %q", c.Name)
	}
	if c.Provider != "Coursera" {
		t.Errorf("provider = %q", c.Provider)
	}
	if c.Institution != "University of Michigan" {
		t.Errorf("institution = %q", c.Institution)
	}
	if c.Rating != 4.8 {
		t.Errorf("rating = %v", c.Rating)
	}
	if c.Reviews != 50000 {
		t.Errorf("reviews = %d", c.Reviews)
	}
	if c.Duration != "4-6 hours/week" {
		t.Errorf("duration = %q", c.Duration)
	}
	if !c.Certificate {
		t.Error("certificate should be true")
	}
	if c.Language != "English" {
		t.Errorf("language = %q", c.Language)
	}
	if c.Level != "Beginner" {
		t.Errorf("level = %q", c.Level)
	}
	if !strings.Contains(c.URL, "python-1") {
		t.Errorf("url = %q, want it to contain slug", c.URL)
	}
}

var sampleSubjectsPage = `<!DOCTYPE html>
<html>
<head></head>
<body>
<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"subjects":[{"name":"Data Science","slug":"data-science","coursesCount":500},{"name":"Programming","slug":"programming","coursesCount":800}]}},"page":"/subjects","query":{}}</script>
</body>
</html>`

func TestExtractSubjectsFromNextData(t *testing.T) {
	subjects := parseSubjects([]byte(sampleSubjectsPage), baseURL)
	if len(subjects) != 2 {
		t.Fatalf("want 2 subjects, got %d", len(subjects))
	}
	if subjects[0].Name != "Data Science" {
		t.Errorf("name = %q", subjects[0].Name)
	}
	if subjects[0].Count != 500 {
		t.Errorf("count = %d", subjects[0].Count)
	}
	if !strings.Contains(subjects[0].URL, "data-science") {
		t.Errorf("url = %q", subjects[0].URL)
	}
	if subjects[0].Rank != 1 {
		t.Errorf("rank = %d", subjects[0].Rank)
	}
}

var sampleProvidersPage = `<!DOCTYPE html>
<html>
<head></head>
<body>
<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"providers":[{"name":"Coursera","slug":"coursera","coursesCount":5000},{"name":"edX","slug":"edx","coursesCount":3000}]}},"page":"/providers","query":{}}</script>
</body>
</html>`

func TestExtractProvidersFromNextData(t *testing.T) {
	providers := parseProviders([]byte(sampleProvidersPage), baseURL)
	if len(providers) != 2 {
		t.Fatalf("want 2 providers, got %d", len(providers))
	}
	if providers[0].Name != "Coursera" {
		t.Errorf("name = %q", providers[0].Name)
	}
	if providers[0].Courses != 5000 {
		t.Errorf("courses = %d", providers[0].Courses)
	}
}

var sampleHTMLCoursePage = `<!DOCTYPE html>
<html>
<body>
<ul>
<li class="course-card-v2">
  <h2 class="course-name"><a href="/course/python-for-everybody">Python for Everybody</a></h2>
  <span class="provider-name">Coursera</span>
  <span class="institution-name">University of Michigan</span>
  <span class="cmpt-rating-number">4.8</span>
</li>
<li class="course-item">
  <h3 class="course-name"><a href="/course/machine-learning">Machine Learning</a></h3>
  <span class="provider-name">Coursera</span>
  <span class="institution-name">Stanford</span>
</li>
</ul>
</body>
</html>`

func TestHtmlCoursesFallback(t *testing.T) {
	courses := htmlCourses([]byte(sampleHTMLCoursePage), baseURL)
	if len(courses) == 0 {
		t.Fatal("expected at least one course from HTML fallback")
	}
	if courses[0].Name == "" {
		t.Error("course name should not be empty")
	}
	if courses[0].URL == "" {
		t.Error("course URL should not be empty")
	}
}

var sampleHTMLSubjectsPage = `<!DOCTYPE html>
<html>
<body>
<div class="subjects-list">
  <a href="/subject/data-science">Data Science</a>
  <a href="/subject/programming">Programming</a>
  <a href="/course/something">Not a subject</a>
</div>
</body>
</html>`

func TestHtmlSubjects(t *testing.T) {
	subjects := htmlSubjects([]byte(sampleHTMLSubjectsPage), baseURL)
	if len(subjects) < 2 {
		t.Fatalf("want at least 2 subjects, got %d", len(subjects))
	}
	names := map[string]bool{}
	for _, s := range subjects {
		names[s.Name] = true
		if !strings.Contains(s.URL, "/subject/") {
			t.Errorf("URL %q does not contain /subject/", s.URL)
		}
	}
	if !names["Data Science"] {
		t.Error("missing Data Science subject")
	}
}

var sampleHTMLProvidersPage = `<!DOCTYPE html>
<html>
<body>
<div class="providers-list">
  <a href="/provider/coursera">Coursera</a>
  <a href="/provider/edx">edX</a>
  <a href="/course/something">Not a provider</a>
</div>
</body>
</html>`

func TestHtmlProviders(t *testing.T) {
	providers := htmlProviders([]byte(sampleHTMLProvidersPage), baseURL)
	if len(providers) < 2 {
		t.Fatalf("want at least 2 providers, got %d", len(providers))
	}
	names := map[string]bool{}
	for _, p := range providers {
		names[p.Name] = true
	}
	if !names["Coursera"] {
		t.Error("missing Coursera provider")
	}
}
