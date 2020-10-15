package awsecs

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPanicMarshal(t *testing.T) {
	type args struct {
		v interface{}
	}
	funcMap := map[string]func(){}
	funcMap["self"] = func() {}
	tests := []struct {
		name    string
		args    args
		wantOut []byte
	}{
		{
			name: "marshal",
			args: args{
				map[string]string{
					"foo": "bar",
				},
			},
			wantOut: []byte(`{"foo":"bar"}`),
		},
		{
			name:    "panic marshal",
			args:    args{funcMap},
			wantOut: []byte("ignored"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if tt.name == "panic marshal" {
					recoverTxt := fmt.Sprint(recover())
					if recoverTxt != "json: unsupported type: func()" {
						t.Error(recoverTxt)
					}
				}
			}()
			if gotOut := panicMarshal(tt.args.v); !reflect.DeepEqual(gotOut, tt.wantOut) {
				t.Errorf("panicMarshal() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestPanicUnmarshal(t *testing.T) {
	type args struct {
		data []byte
		v    interface{}
	}
	simpleMap := map[string]string{}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "unmarshal",
			args: args{
				[]byte(`{"foo":"bar"}`),
				&simpleMap,
			},
		},
		{
			name: "panic unmarshal",
			args: args{
				[]byte("BAD JSON"),
				simpleMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if tt.name == "panic unmarshal" {
					recoverTxt := fmt.Sprint(recover())
					if recoverTxt != "invalid character 'B' looking for beginning of value" {
						t.Error(recoverTxt)
					}
				}
			}()
			panicUnmarshal(tt.args.data, &tt.args.v)
		})
	}
	if simpleMap["foo"] != "bar" {
		t.Error(simpleMap["foo"])
	}
}
