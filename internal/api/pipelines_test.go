package api

import (
	"reflect"
	"testing"
)

func TestPipelineFilterRoundTrip(t *testing.T) {
	base := []string{"Parse CEF", "filter:severity=critical|high|med"}
	filters := pipelineFiltersFromFunctions(base)
	if len(filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(filters))
	}
	merged := mergeFilterFunctions(base, filters)
	if !reflect.DeepEqual(base, merged) {
		t.Fatalf("expected %v, got %v", base, merged)
	}
}

func TestMergeFilterFunctionsAddsStructuredFilters(t *testing.T) {
	base := []string{"Parse CEF", "filter:severity=high"}
	filters := []PipelineFilter{{Field: "severity", Values: []string{"high", "critical"}}}
	merged := mergeFilterFunctions(base, filters)
	want := []string{"Parse CEF", "filter:severity=high|critical"}
	if !reflect.DeepEqual(merged, want) {
		t.Fatalf("want %v, got %v", want, merged)
	}
}
