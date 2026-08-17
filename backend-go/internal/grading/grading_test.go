package grading

import "testing"

func TestGradeObjectiveQuestions(t *testing.T) {
	got := Grade([]Question{{ID: "q1", Type: "multiple_choice", Points: 2, Correct: []string{"B"}}, {ID: "q2", Type: "multiple_select", Points: 3, Correct: []string{"A", "C"}}, {ID: "q3", Type: "true_false", Points: 1, Correct: []string{"true"}}}, []Answer{{QuestionID: "q1", Values: []string{"b"}}, {QuestionID: "q2", Values: []string{"a"}}}, 0.5)
	if got.Earned != 1.5 || got.Correct != 1 || got.Incorrect != 1 || got.Unanswered != 1 {
		t.Fatalf("unexpected result: %+v", got)
	}
}
