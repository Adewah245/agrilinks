package services

import (
	"reflect"
	"testing"
)

func TestWeatherLocationCandidatesUseStructuredFallbacks(t *testing.T) {
	got := weatherLocationCandidates(
		"Obaganya",
		"Dekina",
		"Kogi",
		"Nigeria",
	)

	want := []string{
		"Obaganya",
		"Obaganya, Dekina, Kogi, Nigeria",
		"Dekina, Kogi, Nigeria",
		"Kogi, Nigeria",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("weather location candidates = %#v, want %#v", got, want)
	}
}

func TestWeatherLocationCandidatesRemoveDuplicates(t *testing.T) {
	got := weatherLocationCandidates("Kogi", "", "Kogi", "Nigeria")
	want := []string{"Kogi", "Kogi, Nigeria"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("weather location candidates = %#v, want %#v", got, want)
	}
}
