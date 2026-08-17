package grading

import "strings"

type Question struct {
	ID      string
	Type    string
	Points  float64
	Correct []string
}
type Answer struct {
	QuestionID string
	Values     []string
}
type Outcome struct {
	Earned     float64
	Maximum    float64
	Correct    int
	Incorrect  int
	Unanswered int
}

func Grade(questions []Question, answers []Answer, negativeMarking float64) Outcome {
	byID := map[string][]string{}
	for _, a := range answers {
		byID[a.QuestionID] = normalize(a.Values)
	}
	var out Outcome
	for _, q := range questions {
		out.Maximum += q.Points
		got, exists := byID[q.ID]
		if !exists || len(got) == 0 {
			out.Unanswered++
			continue
		}
		want := normalize(q.Correct)
		if equalSet(got, want) {
			out.Correct++
			out.Earned += q.Points
		} else {
			out.Incorrect++
			out.Earned -= negativeMarking
		}
	}
	if out.Earned < 0 {
		out.Earned = 0
	}
	return out
}
func normalize(v []string) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		out = append(out, strings.ToLower(strings.TrimSpace(x)))
	}
	return out
}
func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]bool{}
	for _, x := range a {
		m[x] = true
	}
	for _, x := range b {
		if !m[x] {
			return false
		}
	}
	return true
}
