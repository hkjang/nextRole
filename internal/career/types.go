// Package career contains NextRole's deterministic, offline career simulation engine.
// Scores are explanatory model outputs, not hiring probabilities or predictions.
package career

import "time"

type Skill struct {
	Name       string  `json:"name"`
	Level      float64 `json:"level"`
	Years      float64 `json:"years"`
	Confidence string  `json:"confidence"`
}

type Profile struct {
	CareerBreakMonths int      `json:"careerBreakMonths"`
	Name              string   `json:"name"`
	CurrentRole       string   `json:"currentRole"`
	YearsExperience   float64  `json:"yearsExperience"`
	Education         string   `json:"education"`
	Region            string   `json:"region"`
	Domain            string   `json:"domain"`
	Narrative         string   `json:"narrative"`
	Skills            []Skill  `json:"skills"`
	Certifications    []string `json:"certifications"`
	Preferences       []string `json:"preferences"`
	WeeklyHours       float64  `json:"weeklyHours"`
}

type Source struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Synthetic bool   `json:"synthetic"`
}

type Requirement struct {
	Name   string  `json:"name"`
	Level  float64 `json:"level"`
	Weight float64 `json:"weight"`
}

type Requirements = []Requirement

type Job struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Category      string        `json:"category"`
	Description   string        `json:"description"`
	Skills        []Requirement `json:"skills"`
	MinExperience float64       `json:"minExperience"`
	Domain        string        `json:"domain"`
	Education     string        `json:"education"`
	Regions       []string      `json:"regions"`
	Source        Source        `json:"source"`
	SalaryRange   string        `json:"salaryRange"`
	BridgeIDs     []string      `json:"bridgeIds"`
}

type Gap struct {
	Name     string  `json:"name"`
	Required float64 `json:"required"`
	Current  float64 `json:"current"`
	Gap      float64 `json:"gap"`
	Weight   float64 `json:"weight"`
}

type Factor struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Max    float64 `json:"max"`
	Reason string  `json:"reason"`
}

type ROI struct {
	Name   string  `json:"name"`
	Before float64 `json:"before"`
	After  float64 `json:"after"`
	Gain   float64 `json:"gain"`
}

type Path struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Roles  []string `json:"roles"`
	Months int      `json:"months"`
	Skills []string `json:"skills"`
	Reuse  float64  `json:"reuse"`
	Cost   float64  `json:"cost"`
	Reason string   `json:"reason"`
}

type Plan struct {
	Month  int      `json:"month"`
	Title  string   `json:"title"`
	Tasks  []string `json:"tasks"`
	Skills []string `json:"skills"`
}

type Simulation struct {
	MarketDemand    int        `json:"marketDemand,omitempty"`
	RankingReason   string     `json:"rankingReason,omitempty"`
	ID              string     `json:"id,omitempty"`
	Job             Job        `json:"job"`
	Score           float64    `json:"score"`
	BaselineScore   float64    `json:"baselineScore"`
	Difficulty      float64    `json:"difficulty"`
	EstimatedMonths int        `json:"estimatedMonths"`
	Gaps            []Gap      `json:"gaps"`
	Factors         []Factor   `json:"factors"`
	ROI             []ROI      `json:"roi"`
	Paths           []Path     `json:"paths"`
	Plan            []Plan     `json:"plan"`
	Strengths       []string   `json:"strengths"`
	Warnings        []string   `json:"warnings"`
	Explanation     string     `json:"explanation"`
	Source          Source     `json:"source"`
	CreatedAt       *time.Time `json:"createdAt,omitempty"`
}

// Weights are normalized to a sum of 100 by the engine. Invalid weights fall
// back to DefaultWeights, so imported configuration cannot produce NaN scores.
type Weights struct {
	Skill      float64 `json:"skill"`
	Transfer   float64 `json:"transfer"`
	Experience float64 `json:"experience"`
	Domain     float64 `json:"domain"`
	Education  float64 `json:"education"`
	Preference float64 `json:"preference"`
}

var DefaultWeights = Weights{40, 20, 15, 10, 5, 10}

type Opportunity struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Organization string   `json:"organization"`
	Region       string   `json:"region"`
	URL          string   `json:"url"`
	Skills       []string `json:"skills"`
	Source       Source   `json:"source"`
	Description  string   `json:"description"`
	Cost         float64  `json:"cost,omitempty"`
	Salary       string   `json:"salary,omitempty"`
	Deadline     string   `json:"deadline,omitempty"`
}
