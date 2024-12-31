package config

import (
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func getEnvName(field reflect.StructField) string {
	return field.Tag.Get("env")
}

func setFieldFromEnv(field reflect.Value, envName string) {
	if envName == "" {
		return
	}

	// Check if env var is actually set
	envValue, exists := os.LookupEnv(envName)
	if !exists {
		return
	}

	switch field.Kind() {

	case reflect.String:
		field.SetString(envValue)

	case reflect.Bool:
		if val, err := strconv.ParseBool(envValue); err == nil {
			field.SetBool(val)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, err := strconv.ParseInt(envValue, 10, 64); err == nil {
			field.SetInt(val)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(envValue, 10, 64); err == nil {
			field.SetUint(val)
		}

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			field.Set(reflect.ValueOf(strings.Split(envValue, ",")))
		}
	}
}

func processStruct(val reflect.Value) {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		typeField := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs recursively
		if field.Kind() == reflect.Struct {
			processStruct(field)
			continue
		}

		envName := getEnvName(typeField)
		setFieldFromEnv(field, envName)
	}
}

func overlayEnvVars(cfg *Plugin) error {
	// Special handling for Gotify URL
	if urlStr, exists := os.LookupEnv("TG_PLUGIN__GOTIFY_URL"); exists {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return err
		}

		cfg.Settings.GotifyServer.RawUrl = urlStr
		cfg.Settings.GotifyServer.Url = parsedURL
	}

	// Process all other fields dynamically
	val := reflect.ValueOf(cfg).Elem()
	processStruct(val)

	return nil
}
