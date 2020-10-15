package awsecs

import "encoding/json"

func panicMarshal(v interface{}) (out []byte) {
	out, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return
}

func panicUnmarshal(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}
