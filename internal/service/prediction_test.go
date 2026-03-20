// internal/service/prediction_test.go
package service

import "testing"

func TestCalculateRiskScore(t *testing.T) {
	tests := []struct {
		name       string
		cvss       float64
		hasExploit bool
		age        int
		minScore   float64
		maxScore   float64
	}{
		{"critical with exploit", 9.8, true, 30, 90.0, 100.0},
		{"medium no exploit", 5.0, false, 10, 30.0, 60.0},
		{"low old", 2.0, false, 365, 10.0, 35.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateRiskScore(tt.cvss, tt.hasExploit, tt.age)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("score %.2f outside expected range [%.0f, %.0f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}
