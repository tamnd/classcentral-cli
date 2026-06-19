package classcentral

import "fmt"

// Course is the record emitted for a Class Central course.
type Course struct {
	Rank        int     `json:"rank"        table:"RANK"`
	Name        string  `json:"name"        table:"NAME"`
	Provider    string  `json:"provider"    table:"PROVIDER"`
	Institution string  `json:"institution" table:"INSTITUTION"`
	Rating      float64 `json:"rating"      table:"RATING"`
	Reviews     int     `json:"reviews"     table:"REVIEWS"`
	Enrollments int     `json:"enrollments" table:"ENROLLED"`
	Level       string  `json:"level"       table:"LEVEL"`
	Language    string  `json:"language"    table:"LANG"`
	Certificate bool    `json:"certificate" table:"CERT"`
	Duration    string  `json:"duration"    table:"DURATION"`
	URL         string  `json:"url"         table:"URL"`
}

// Subject is the record emitted for a Class Central subject category.
type Subject struct {
	Rank  int    `json:"rank"  table:"RANK"`
	Name  string `json:"name"  table:"NAME"`
	Count int    `json:"count" table:"COURSES"`
	URL   string `json:"url"   table:"URL"`
}

// Provider is the record emitted for a course platform indexed by Class Central.
type Provider struct {
	Rank    int    `json:"rank"    table:"RANK"`
	Name    string `json:"name"    table:"NAME"`
	Courses int    `json:"courses" table:"COURSES"`
	URL     string `json:"url"     table:"URL"`
}

// ─── REST API wire types ──────────────────────────────────────────────────────

type searchResp struct {
	Courses    []wireCourse `json:"courses"`
	TotalCount int          `json:"totalCount"`
}

type subjectsResp struct {
	Subjects []wireSubject `json:"subjects"`
}

type providersResp struct {
	Providers []wireProvider `json:"providers"`
}

type wireCourse struct {
	ID              int            `json:"id"`
	Name            string         `json:"name"`
	Slug            string         `json:"slug"`
	Rating          float64        `json:"rating"`
	ReviewCount     int            `json:"review_count"`
	EnrollmentCount int            `json:"enrollment_count"`
	DurationWeeks   int            `json:"duration_in_weeks"`
	HoursPerWeek    int            `json:"hours_per_week"`
	Certificate     bool           `json:"certificate"`
	Language        string         `json:"language"`
	Level           string         `json:"level"`
	Providers       []wireProvider `json:"providers"`
	Universities    []wireUniv     `json:"universities"`
	Subjects        []wireSubjTag  `json:"subjects"`
	URL             string         `json:"url"`
}

type wireProvider struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	CourseCount int    `json:"course_count"`
}

type wireSubject struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	CourseCount int    `json:"course_count"`
}

type wireUniv struct {
	Name string `json:"name"`
}

type wireSubjTag struct {
	Name string `json:"name"`
}

func (w wireCourse) toCourse(rank int) Course {
	provider := ""
	if len(w.Providers) > 0 {
		provider = w.Providers[0].Name
	}
	institution := ""
	if len(w.Universities) > 0 {
		institution = w.Universities[0].Name
	}
	duration := ""
	if w.DurationWeeks > 0 && w.HoursPerWeek > 0 {
		duration = fmt.Sprintf("%d weeks, %d hrs/week", w.DurationWeeks, w.HoursPerWeek)
	} else if w.DurationWeeks > 0 {
		duration = fmt.Sprintf("%d weeks", w.DurationWeeks)
	}
	u := w.URL
	if u == "" && w.Slug != "" {
		u = "https://www.classcentral.com/course/" + w.Slug
	}
	return Course{
		Rank:        rank,
		Name:        w.Name,
		Provider:    provider,
		Institution: institution,
		Rating:      w.Rating,
		Reviews:     w.ReviewCount,
		Enrollments: w.EnrollmentCount,
		Level:       w.Level,
		Language:    w.Language,
		Certificate: w.Certificate,
		Duration:    duration,
		URL:         u,
	}
}

// ─── __NEXT_DATA__ wire types (HTML fallback, used by parse.go) ───────────────

type ndCourse struct {
	ID               int             `json:"id"`
	Name             string          `json:"name"`
	Slug             string          `json:"slug"`
	Provider         ndProvider      `json:"provider"`
	Institutions     []ndInstitution `json:"institutions"`
	Rating           float64         `json:"rating"`
	ReviewsCount     int             `json:"reviewsCount"`
	EnrollmentsCount int             `json:"enrollmentsCount"`
	Workload         string          `json:"workload"`
	Length           string          `json:"length"`
	Certificate      bool            `json:"certificate"`
	Language         string          `json:"language"`
	Level            string          `json:"level"`
}

type ndProvider struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ndInstitution struct {
	Name string `json:"name"`
}

type ndSubject struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Count int    `json:"coursesCount"`
}

type ndProviderWire struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	CoursesCount int    `json:"coursesCount"`
}

// ─── __NEXT_DATA__ conversion helpers ────────────────────────────────────────

func courseFromND(c ndCourse, rank int, baseURL string) Course {
	inst := ""
	if len(c.Institutions) > 0 {
		inst = c.Institutions[0].Name
	}
	dur := c.Workload
	if dur == "" {
		dur = c.Length
	}
	u := baseURL + "/course/" + c.Slug
	return Course{
		Rank:        rank,
		Name:        c.Name,
		Provider:    c.Provider.Name,
		Institution: inst,
		Rating:      c.Rating,
		Reviews:     c.ReviewsCount,
		Enrollments: c.EnrollmentsCount,
		Duration:    dur,
		Certificate: c.Certificate,
		Language:    c.Language,
		Level:       c.Level,
		URL:         u,
	}
}

func subjectFromND(s ndSubject, rank int, baseURL string) Subject {
	return Subject{
		Rank:  rank,
		Name:  s.Name,
		Count: s.Count,
		URL:   baseURL + "/subject/" + s.Slug,
	}
}

func providerFromND(p ndProviderWire, rank int, baseURL string) Provider {
	return Provider{
		Rank:    rank,
		Name:    p.Name,
		Courses: p.CoursesCount,
		URL:     baseURL + "/provider/" + p.Slug,
	}
}
