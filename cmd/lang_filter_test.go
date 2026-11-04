package cmd

import (
	"reflect"
	"testing"

	"github.com/jiotv-go/jiotv_go/v3/internal/config"
)

func TestApplyLangFilter(t *testing.T) {
	tests := []struct {
		name    string
		start   []int
		value   string
		want    []int
		wantErr bool
	}{
		{"not given keeps config", []int{5}, "", []int{5}, false},
		{"0 shows all", []int{1, 6}, "0", nil, false},
		{"1 is Hindi and English", nil, "1", []int{1, 6}, false},
		{"custom list", nil, " 1, 6,5 ", []int{1, 6, 5}, false},
		{"bad value", []int{5}, "hindi", []int{5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Cfg.DefaultLanguages = tt.start
			err := ApplyLangFilter(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(config.Cfg.DefaultLanguages, tt.want) {
				t.Errorf("DefaultLanguages = %v, want %v", config.Cfg.DefaultLanguages, tt.want)
			}
		})
	}
}
