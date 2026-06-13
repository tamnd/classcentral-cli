package classcentral

// Course is the record emitted for a Class Central course.
type Course struct {
	Rank        int     `json:"rank"`
	Name        string  `json:"name"`
	Provider    string  `json:"provider"`
	Institution string  `json:"institution"`
	Rating      float64 `json:"rating"`
	Reviews     int     `json:"reviews"`
	Enrollments int     `json:"enrollments"`
	Duration    string  `json:"duration"`
	Certificate bool    `json:"certificate"`
	Language    string  `json:"language"`
	Level       string  `json:"level"`
	URL         string  `json:"url"`
}

// Subject is the record emitted for a Class Central subject category.
type Subject struct {
	Rank  int    `json:"rank"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	URL   string `json:"url"`
}

// Provider is the record emitted for a course platform indexed by Class Central.
type Provider struct {
	Rank    int    `json:"rank"`
	Name    string `json:"name"`
	Courses int    `json:"courses"`
	URL     string `json:"url"`
}

// ─── wire types from __NEXT_DATA__ ───────────────────────────────────────────

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

// ─── conversion helpers ───────────────────────────────────────────────────────

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
