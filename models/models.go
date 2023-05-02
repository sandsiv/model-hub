package models

type ModelName string
type Model struct {
	Name ModelName
	Path string
}

type Instance struct {
	Value string `json:"value"`
}

type Parameter struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type PredictRequest struct {
	Instances []interface{}          `json:"instances"`
	Params    map[string]interface{} `json:"parameters"`
}

type PredictResponse struct {
	Predictions []interface{} `json:"predictions"`
}
