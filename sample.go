package mocjson

type SampleObject1 struct {
	Boolean      bool            `json:"boolean"`
	Float64      float64         `json:"float64"`
	String       string          `json:"string"`
	Object       map[string]any  `json:"object"`
	Array        []any           `json:"array"`
	Any          any             `json:"any"`
	Object2      SampleObject2   `json:"object2"`
	Object2Array []SampleObject2 `json:"object2_array"`
}

type SampleObject2 struct {
	Float64 float64        `json:"float64"`
	String  string         `json:"string"`
	Boolean bool           `json:"boolean"`
	Object  map[string]any `json:"object"`
	Array   []any          `json:"array"`
	Any     any            `json:"any"`
}
