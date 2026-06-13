package classcentral

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// nextDataRe matches the __NEXT_DATA__ script tag and captures its JSON content.
var nextDataRe = regexp.MustCompile(`(?s)<script\s+id="__NEXT_DATA__"[^>]*>(\{.*?\})</script>`)

// nextData extracts and decodes the __NEXT_DATA__ JSON from a page body.
// Returns nil, nil when the tag is absent.
func nextData(body []byte) (map[string]any, error) {
	m := nextDataRe.FindSubmatch(body)
	if m == nil {
		return nil, nil
	}
	var nd map[string]any
	if err := json.Unmarshal(m[1], &nd); err != nil {
		return nil, err
	}
	return nd, nil
}

// pageProps walks a nextData map to props.pageProps.
func pageProps(nd map[string]any) map[string]any {
	props, _ := nd["props"].(map[string]any)
	if props == nil {
		return nil
	}
	pp, _ := props["pageProps"].(map[string]any)
	return pp
}

// parseCourses extracts courses from body using __NEXT_DATA__ first,
// falling back to HTML card parsing.
func parseCourses(body []byte, baseURL string) []Course {
	nd, err := nextData(body)
	if err == nil && nd != nil {
		pp := pageProps(nd)
		if pp != nil {
			if courses := extractNDCourses(pp, baseURL); courses != nil {
				return courses
			}
		}
	}
	return htmlCourses(body, baseURL)
}

// extractNDCourses reads courses from pageProps. Returns nil when absent.
func extractNDCourses(pp map[string]any, baseURL string) []Course {
	raw, ok := pp["courses"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]Course, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := ndCourseFromMap(m)
		out = append(out, courseFromND(c, i+1, baseURL))
	}
	return out
}

// parseSubjects extracts subjects from body.
func parseSubjects(body []byte, baseURL string) []Subject {
	nd, err := nextData(body)
	if err == nil && nd != nil {
		pp := pageProps(nd)
		if pp != nil {
			if subjects := extractNDSubjects(pp, baseURL); subjects != nil {
				return subjects
			}
		}
	}
	return htmlSubjects(body, baseURL)
}

func extractNDSubjects(pp map[string]any, baseURL string) []Subject {
	raw, ok := pp["subjects"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]Subject, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		s := ndSubject{
			Name:  strVal(m, "name"),
			Slug:  strVal(m, "slug"),
			Count: intVal(m, "coursesCount"),
		}
		if s.Count == 0 {
			s.Count = intVal(m, "count")
		}
		out = append(out, subjectFromND(s, i+1, baseURL))
	}
	return out
}

// parseProviders extracts providers from body.
func parseProviders(body []byte, baseURL string) []Provider {
	nd, err := nextData(body)
	if err == nil && nd != nil {
		pp := pageProps(nd)
		if pp != nil {
			if providers := extractNDProviders(pp, baseURL); providers != nil {
				return providers
			}
		}
	}
	return htmlProviders(body, baseURL)
}

func extractNDProviders(pp map[string]any, baseURL string) []Provider {
	raw, ok := pp["providers"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]Provider, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		p := ndProviderWire{
			Name:         strVal(m, "name"),
			Slug:         strVal(m, "slug"),
			CoursesCount: intVal(m, "coursesCount"),
		}
		if p.CoursesCount == 0 {
			p.CoursesCount = intVal(m, "count")
		}
		out = append(out, providerFromND(p, i+1, baseURL))
	}
	return out
}

// ndCourseFromMap converts a generic map (from JSON unmarshalling into any)
// into an ndCourse.
func ndCourseFromMap(m map[string]any) ndCourse {
	c := ndCourse{
		Name:             strVal(m, "name"),
		Slug:             strVal(m, "slug"),
		Rating:           floatVal(m, "rating"),
		ReviewsCount:     intVal(m, "reviewsCount"),
		EnrollmentsCount: intVal(m, "enrollmentsCount"),
		Workload:         strVal(m, "workload"),
		Length:           strVal(m, "length"),
		Language:         strVal(m, "language"),
		Level:            strVal(m, "level"),
	}
	if v, ok := m["certificate"]; ok {
		c.Certificate, _ = v.(bool)
	}
	if pv, ok := m["provider"].(map[string]any); ok {
		c.Provider = ndProvider{
			Name: strVal(pv, "name"),
			Slug: strVal(pv, "slug"),
		}
	}
	if iv, ok := m["institutions"].([]any); ok {
		for _, inst := range iv {
			if im, ok := inst.(map[string]any); ok {
				c.Institutions = append(c.Institutions, ndInstitution{Name: strVal(im, "name")})
			}
		}
	}
	return c
}

// ─── HTML fallback parsers ────────────────────────────────────────────────────

// htmlCourses parses course cards from raw HTML using the x/net/html tokenizer.
// This is the fallback when __NEXT_DATA__ is absent or incomplete.
func htmlCourses(body []byte, baseURL string) []Course {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var courses []Course
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			cls := attrVal(n, "class")
			// Class Central marks course cards with "course-card" or similar classes
			if n.Data == "li" && (strings.Contains(cls, "course-card") || strings.Contains(cls, "course-item")) {
				if c := parseCourseCard(n, baseURL, len(courses)+1); c != nil {
					courses = append(courses, *c)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return courses
}

func parseCourseCard(n *html.Node, baseURL string, rank int) *Course {
	c := &Course{Rank: rank}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			cls := attrVal(node, "class")
			switch {
			case node.Data == "h2" && strings.Contains(cls, "course-name"),
				node.Data == "h3" && strings.Contains(cls, "course-name"),
				node.Data == "h4" && strings.Contains(cls, "course-name"):
				if a := findA(node); a != nil {
					c.Name = textContent(a)
					href := attrVal(a, "href")
					if href != "" {
						if strings.HasPrefix(href, "http") {
							c.URL = href
						} else {
							c.URL = baseURL + href
						}
					}
				}
			case strings.Contains(cls, "provider-name") || strings.Contains(cls, "institution-name"):
				if c.Provider == "" {
					c.Provider = textContent(node)
				} else {
					c.Institution = textContent(node)
				}
			case strings.Contains(cls, "rating") && strings.Contains(cls, "number"):
				r, _ := strconv.ParseFloat(strings.TrimSpace(textContent(node)), 64)
				c.Rating = r
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	if c.Name == "" {
		return nil
	}
	return c
}

// htmlSubjects parses subject links from the /subjects page HTML.
func htmlSubjects(body []byte, baseURL string) []Subject {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var subjects []Subject
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attrVal(n, "href")
			if strings.Contains(href, "/subject/") {
				name := textContent(n)
				name = strings.TrimSpace(name)
				if name != "" {
					u := href
					if !strings.HasPrefix(href, "http") {
						u = baseURL + href
					}
					subjects = append(subjects, Subject{
						Rank: len(subjects) + 1,
						Name: name,
						URL:  u,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return dedupSubjects(subjects)
}

// htmlProviders parses provider cards from the /providers page HTML.
func htmlProviders(body []byte, baseURL string) []Provider {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	var providers []Provider
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attrVal(n, "href")
			if strings.Contains(href, "/provider/") {
				name := textContent(n)
				name = strings.TrimSpace(name)
				if name != "" {
					u := href
					if !strings.HasPrefix(href, "http") {
						u = baseURL + href
					}
					providers = append(providers, Provider{
						Rank: len(providers) + 1,
						Name: name,
						URL:  u,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return dedupProviders(providers)
}

// ─── HTML helpers ─────────────────────────────────────────────────────────────

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}

func findA(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "a" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if a := findA(c); a != nil {
			return a
		}
	}
	return nil
}

func dedupSubjects(ss []Subject) []Subject {
	seen := map[string]bool{}
	out := ss[:0]
	for _, s := range ss {
		if !seen[s.URL] {
			seen[s.URL] = true
			out = append(out, s)
		}
	}
	return out
}

func dedupProviders(pp []Provider) []Provider {
	seen := map[string]bool{}
	out := pp[:0]
	for _, p := range pp {
		if !seen[p.URL] {
			seen[p.URL] = true
			out = append(out, p)
		}
	}
	return out
}

// ─── map helpers ──────────────────────────────────────────────────────────────

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func floatVal(m map[string]any, key string) float64 {
	v, _ := m[key].(float64)
	return v
}
