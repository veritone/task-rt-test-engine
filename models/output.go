package models

type SeriesObject struct {
	Start     int32  `json:"startTimeMs"`
	End       int32  `json:"stopTimeMs"`
	EntityID  string `json:"entityId"`
	LibraryID string `json:"libraryId"`
	Object    Object `json:"object"`
}

type Object struct {
	Label        string   `json:"label"`
	ObjectType   string   `json:"type"`
	Confidence   float64  `json:"confidence"`
}