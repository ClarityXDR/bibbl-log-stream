package secrets

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Resolver defines the minimal capability required to hydrate secret references.
type Resolver interface {
	Resolve(ctx context.Context, ref string) (string, error)
}

// ReplacePlaceholders walks the provided structure and replaces any string values
// that reference a supported placeholder (currently vault://) via the supplied resolver.
func ReplacePlaceholders(ctx context.Context, target interface{}, resolver Resolver) error {
	if target == nil || resolver == nil {
		return nil
	}
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return errors.New("target must be a non-nil pointer")
	}
	return walkValue(ctx, val.Elem(), resolver)
}

func walkValue(ctx context.Context, val reflect.Value, resolver Resolver) error {
	switch val.Kind() {
	case reflect.String:
		if !val.CanSet() {
			return nil
		}
		raw := val.String()
		if newVal, changed, err := maybeResolve(ctx, raw, resolver); err != nil {
			return err
		} else if changed {
			val.SetString(newVal)
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if !field.CanSet() && field.Kind() != reflect.Map && field.Kind() != reflect.Slice && field.Kind() != reflect.Array && field.Kind() != reflect.Interface && field.Kind() != reflect.Pointer {
				continue
			}
			if err := walkValue(ctx, field, resolver); err != nil {
				return err
			}
		}
	case reflect.Pointer:
		if !val.IsNil() {
			return walkValue(ctx, val.Elem(), resolver)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			if elem.Kind() == reflect.String {
				if newVal, changed, err := maybeResolve(ctx, elem.String(), resolver); err != nil {
					return err
				} else if changed {
					elem.SetString(newVal)
				}
				continue
			}
			if elem.CanAddr() || elem.CanSet() {
				if err := walkValue(ctx, elem, resolver); err != nil {
					return err
				}
				continue
			}
			tmp := reflect.New(elem.Type()).Elem()
			tmp.Set(elem)
			if err := walkValue(ctx, tmp, resolver); err != nil {
				return err
			}
			elem.Set(tmp)
		}
	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			key := iter.Key()
			value := iter.Value()
			if value.Kind() == reflect.String {
				newVal, changed, err := maybeResolve(ctx, value.String(), resolver)
				if err != nil {
					return err
				}
				if changed {
					val.SetMapIndex(key, reflect.ValueOf(newVal).Convert(value.Type()))
				}
				continue
			}
			tmp := reflect.New(value.Type()).Elem()
			tmp.Set(value)
			if err := walkValue(ctx, tmp, resolver); err != nil {
				return err
			}
			val.SetMapIndex(key, tmp)
		}
	case reflect.Interface:
		if val.IsNil() {
			return nil
		}
		inner := val.Elem()
		tmp := reflect.New(inner.Type()).Elem()
		tmp.Set(inner)
		if err := walkValue(ctx, tmp, resolver); err != nil {
			return err
		}
		val.Set(tmp)
	}
	return nil
}

func maybeResolve(ctx context.Context, raw string, resolver Resolver) (string, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false, nil
	}
	if !strings.HasPrefix(trimmed, "vault://") {
		return raw, false, nil
	}
	val, err := resolver.Resolve(ctx, trimmed)
	if err != nil {
		return "", false, fmt.Errorf("resolve %s: %w", trimmed, err)
	}
	return val, true, nil
}
