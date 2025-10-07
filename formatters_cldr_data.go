// Code generated for staging CLDR bundles. DO NOT EDIT.

package i18n

type cldrListPatterns struct {
	Pair   string
	Start  string
	Middle string
	End    string
}

type cldrOrdinalRules struct {
	System string
}

type cldrMeasurementData struct {
	Units map[string]string
}

type cldrPhoneMetadata struct {
	CountryCode    string
	NationalPrefix string
	Groups         []int
}

type cldrBundle struct {
	List        cldrListPatterns
	Ordinal     cldrOrdinalRules
	Measurement cldrMeasurementData
	Phone       cldrPhoneMetadata
}

var cldrBundles = map[string]cldrBundle{
	"en": {
		List: cldrListPatterns{
			Pair:   "{0} and {1}",
			Start:  "{0}, {1}",
			Middle: "{0}, {1}",
			End:    "{0}, and {1}",
		},
		Ordinal: cldrOrdinalRules{
			System: "english",
		},
		Measurement: cldrMeasurementData{
			Units: map[string]string{
				"km": "km",
				"kg": "kg",
				"m":  "m",
				"mi": "mi",
				"lb": "lb",
			},
		},
		Phone: cldrPhoneMetadata{
			CountryCode:    "1",
			NationalPrefix: "1",
			Groups:         []int{3, 3, 4},
		},
	},
	"es": {
		List: cldrListPatterns{
			Pair:   "{0} y {1}",
			Start:  "{0}, {1}",
			Middle: "{0}, {1}",
			End:    "{0} y {1}",
		},
		Ordinal: cldrOrdinalRules{
			System: "spanish",
		},
		Measurement: cldrMeasurementData{
			Units: map[string]string{
				"km": "km",
				"kg": "kg",
				"m":  "m",
				"mi": "millas",
				"lb": "lb",
			},
		},
		Phone: cldrPhoneMetadata{
			CountryCode:    "34",
			NationalPrefix: "",
			Groups:         []int{3, 3, 3},
		},
	},
}
