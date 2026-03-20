// internal/service/prediction.go
package service

import "math"

// CalculateRiskScore produces a 0-100 risk score based on:
// - CVSS score (0-10, weighted 60%)
// - Exploit availability (boolean, weighted 25%)
// - Age in days (weighted 15% — older unpatched = higher risk)
func CalculateRiskScore(cvss float64, hasExploit bool, ageDays int) float64 {
	cvssComponent := (cvss / 10.0) * 60.0

	exploitComponent := 0.0
	if hasExploit {
		exploitComponent = 25.0
	}

	// Age factor: logarithmic curve, caps contribution at 15
	ageFactor := math.Min(math.Log1p(float64(ageDays))/math.Log1p(365)*15.0, 15.0)

	score := cvssComponent + exploitComponent + ageFactor
	return math.Min(math.Max(score, 0), 100)
}
