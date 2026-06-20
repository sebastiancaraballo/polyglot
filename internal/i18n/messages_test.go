package i18n

import (
	"reflect"
	"strings"
	"testing"
)

func TestSpanishMessagesUseLanguageCodes(t *testing.T) {
	if ES.Tagline != "es → ja" {
		t.Fatalf("Tagline = %q, want %q", ES.Tagline, "es → ja")
	}
}

func TestSpanishMessagesAvoidPictographicEmoji(t *testing.T) {
	banned := []string{
		"\U0001F1EA\U0001F1F8", // Spain flag.
		"\U0001F1EF\U0001F1F5", // Japan flag.
		"\U0001F3B4",           // flower playing cards.
		"\U0001F4CA",           // bar chart.
		"\U0001F525",           // fire.
		"\U0001F389",           // party popper.
		"\U0001F319",           // crescent moon.
		"\u2728",               // sparkles.
		"\U0001F464",           // bust silhouette.
		"\u267F",               // wheelchair symbol.
	}
	values := messageStrings(reflect.ValueOf(ES))
	for _, value := range values {
		for _, emoji := range banned {
			if strings.Contains(value, emoji) {
				t.Fatalf("message %q contains banned emoji %q", value, emoji)
			}
		}
	}
}

func messageStrings(v reflect.Value) []string {
	var values []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		switch field.Kind() {
		case reflect.String:
			values = append(values, field.String())
		case reflect.Slice:
			if field.Type().Elem().Kind() != reflect.String {
				continue
			}
			for j := 0; j < field.Len(); j++ {
				values = append(values, field.Index(j).String())
			}
		}
	}
	return values
}
