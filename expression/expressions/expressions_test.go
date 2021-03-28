package expressions

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/xtoproto/expression"
)

func TestReflectAssumptions(t *testing.T) {
	aString := ""
	aStringPtr := &aString
	tests := []struct {
		name      string
		got, want interface{}
	}{
		{
			name: "string to *string - not assignable",
			got:  reflect.TypeOf("abc").AssignableTo(reflect.TypeOf(aStringPtr)),
			want: false,
		},
		{
			name: "string to *string - assignable",
			got:  reflect.TypeOf("abc").AssignableTo(reflect.TypeOf(aStringPtr).Elem()),
			want: true,
		},
		{
			name: "*string.Kind is Ptr",
			got:  reflect.TypeOf(aStringPtr).Kind(),
			want: reflect.Ptr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.got); diff != "" {
				t.Errorf("assumption failed (-want, +got):\n  %s", diff)
			}
		})
	}
}

func TestBind(t *testing.T) {
	type ab struct {
		A string
		B string
	}

	type args struct {
		exp *expression.Expression
		dst interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "string to *string",
			args: args{
				exp: MustParse(`"a"`),
				dst: func() *string { var s string; return &s }(),
			},
			want:    func() *string { s := "a"; return &s }(),
			wantErr: false,
		},
		{
			name: "123 to int",
			args: args{
				exp: MustParse(`123`),
				dst: new(int),
			},
			want:    intPtr(123),
			wantErr: false,
		},
		{
			name: "123 to int",
			args: args{
				exp: MustParse(`("abc" "123")`),
				dst: new(ab),
			},
			want:    &ab{"abc", "123"},
			wantErr: false,
		},
		{
			name: "123 to int",
			args: args{
				exp: MustParse(`("abc" 123)`),
				dst: new(ab),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Bind(tt.args.exp, tt.args.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, tt.args.dst); diff != "" {
				t.Errorf("Bind() produced unexpected result (-want, +got):\n  %s", diff)
			}
		})
	}
}

func stringPtr(v string) *string { return &v }
func uintPtr(v uint) *uint       { return &v }
func uint8Ptr(v uint8) *uint8    { return &v }
func uint16Ptr(v uint16) *uint16 { return &v }
func uint32Ptr(v uint32) *uint32 { return &v }
func uint64Ptr(v uint64) *uint64 { return &v }
func intPtr(v int) *int          { return &v }
func int8Ptr(v int8) *int8       { return &v }
func int16Ptr(v int16) *int16    { return &v }
func int32Ptr(v int32) *int32    { return &v }
func int64Ptr(v int64) *int64    { return &v }
