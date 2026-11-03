package services

import (
	"strings"
	"unicode"
)

// Open-Meteo indexes populated places rather than state boundaries. When a
// community and LGA cannot be resolved, look up the state's capital and label
// that broader estimate explicitly. Coordinates still come from the provider
// and must match the selected state.
var weatherStateCapitals = map[string]string{
	"abia": "Umuahia", "adamawa": "Yola", "akwa ibom": "Uyo",
	"anambra": "Awka", "bauchi": "Bauchi", "bayelsa": "Yenagoa",
	"benue": "Makurdi", "borno": "Maiduguri", "cross river": "Calabar",
	"delta": "Asaba", "ebonyi": "Abakaliki", "edo": "Benin City",
	"ekiti": "Ado Ekiti", "enugu": "Enugu", "fct": "Abuja",
	"gombe": "Gombe", "imo": "Owerri", "jigawa": "Dutse",
	"kaduna": "Kaduna", "kano": "Kano", "katsina": "Katsina",
	"kebbi": "Birnin Kebbi", "kogi": "Lokoja", "kwara": "Ilorin",
	"lagos": "Ikeja", "nasarawa": "Lafia", "niger": "Minna",
	"ogun": "Abeokuta", "ondo": "Akure", "osun": "Osogbo",
	"oyo": "Ibadan", "plateau": "Jos", "rivers": "Port Harcourt",
	"sokoto": "Sokoto", "taraba": "Jalingo", "yobe": "Damaturu",
	"zamfara": "Gusau",
}

func normalizeWeatherLGA(lga string) string {
	lga = normalizeWeatherLocation(lga)
	for _, suffix := range []string{" local government area", " area council", " lga"} {
		lga = strings.TrimSuffix(lga, suffix)
	}
	lga = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, lga)
	switch lga {
	case "municipal", "abujamunicipal", "amac":
		return "abujamunicipal"
	}
	return lga
}
