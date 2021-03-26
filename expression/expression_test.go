package expression

import (
	"reflect"
	"testing"
)

func TestParseSExpression(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    *Expression
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSExpression(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
