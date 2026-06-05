package verify

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	fxReg = regexp.MustCompile(`^(\w+)\((.*?)\)$`)
)

const (
	MIN      = "min"
	MAX      = "max"
	RANGE    = "range"
	LEN      = "len"
	REGEXP   = "regexp"
	CONTAIN  = "contain"
	EXCLUDE  = "exclude"
	CUSTOM   = "custom"
	REQUIRED = "required"
)

type IVerify interface {
	CustomVerify() VerifyFuncMap
}

type (
	VerifyFunc    func(string) error
	VerifyFuncMap map[string]VerifyFunc
)

var customMap = make(VerifyFuncMap)

func SetCustomVerifyFunc(name string, fn VerifyFunc) {
	customMap[name] = fn
}

func Verify(v any) error {

	vt := reflect.TypeOf(v)
	vv := reflect.ValueOf(v)

	verify, ok := v.(IVerify)
	if ok {
		customMap = verify.CustomVerify()
	}

	for vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
		vv = vv.Elem()
	}

	for i := 0; i < vt.NumField(); i++ {
		field := vt.Field(i)
		value := vv.Field(i)
		for _, fx := range strings.Split(field.Tag.Get("verify"), ";") {
			if fxReg.MatchString(strings.TrimSpace(fx)) {
				match := fxReg.FindStringSubmatch(strings.TrimSpace(fx))[1:]
				switch match[0] {
				case MIN:
					minSize, _ := strconv.Atoi(match[1])
					if err := fxMin(field, value, int64(minSize)); err != nil {
						return err
					}
				case MAX:
					maxSize, _ := strconv.Atoi(match[1])
					if err := fxMax(field, value, int64(maxSize)); err != nil {
						return err
					}
				case RANGE:
					args := strings.Split(match[1], ",")
					if len(args) == 2 {
						minSize, _ := strconv.Atoi(args[0])
						maxSize, _ := strconv.Atoi(args[1])
						if err := fxMin(field, value, int64(minSize)); err != nil {
							return err
						}
						if err := fxMax(field, value, int64(maxSize)); err != nil {
							return err
						}
					}
				case LEN:
					lenSize, _ := strconv.Atoi(match[1])
					if err := fxLen(field, value, int64(lenSize)); err != nil {
						return err
					}
				case REGEXP:
					if err := fxRegexp(field, value, match[1]); err != nil {
						return err
					}
				case CONTAIN:
					items := strings.Split(match[1], ",")
					if err := fxContain(field, value, items); err != nil {
						return err
					}
				case EXCLUDE:
					items := strings.Split(match[1], ",")
					if err := fxExclude(field, value, items); err != nil {
						return err
					}
				case CUSTOM:
					fn, ok := customMap[match[1]]
					if ok {
						if err := fn(fmt.Sprintf("%v", value)); err != nil {
							return fmt.Errorf("%v: %v", field.Name, err)
						}
					}
				case REQUIRED:
					if err := fxRequired(field, value, match); err != nil {
						return err
					}
				default:

				}
			}
		}
	}
	return nil
}

func fxMin(field reflect.StructField, value reflect.Value, size int64) (err error) {
	if compare(value, float64(size)) < 0 {
		err = fmt.Errorf("the length of %v is less than %v", field.Name, size)
	}
	return
}

func fxMax(field reflect.StructField, value reflect.Value, size int64) (err error) {
	if compare(value, float64(size)) > 0 {
		err = fmt.Errorf("the length of %v is greater than than %v", field.Name, size)
	}
	return
}

func fxLen(field reflect.StructField, value reflect.Value, size int64) (err error) {
	if compare(value, float64(size)) != 0 {
		err = fmt.Errorf("the %v length is not %v", field.Name, size)
	}
	return
}

func compare(value reflect.Value, size float64) float64 {
	var _len float64
	switch value.Type().Kind() {
	case reflect.String:
		_len = float64(utf8.RuneCountInString(fmt.Sprintf("%v", value.Interface())))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		_len = float64(value.Int())
	case reflect.Float32, reflect.Float64:
		_len = value.Float()
	case reflect.Array, reflect.Slice:
		_len = float64(value.Len())
	default:
		return 0
	}
	return _len - size
}

func fxRegexp(field reflect.StructField, value reflect.Value, str string) (err error) {
	sprintf := fmt.Sprintf("%v", value.Interface())
	if !regexp.MustCompile(str).MatchString(sprintf) {
		err = fmt.Errorf("%v cannot match the regular expression '%v'", field.Name, str)
	}
	return
}

func fxContain(field reflect.StructField, value reflect.Value, items []string) (err error) {
	if !slices.Contains(items, fmt.Sprintf("%v", value.Interface())) {
		err = fmt.Errorf("%v exclude %v", strings.Join(items, ", "), field.Name)
	}
	return
}

func fxExclude(field reflect.StructField, value reflect.Value, items []string) (err error) {
	if slices.Contains(items, fmt.Sprintf("%v", value.Interface())) {
		err = fmt.Errorf("%v contains %v", strings.Join(items, ", "), field.Name)
	}
	return
}

func fxRequired(field reflect.StructField, value reflect.Value, items []string) (err error) {
	if len(strings.TrimSpace(fmt.Sprintf("%v", value.Interface()))) == 0 {
		err = fmt.Errorf("%v is required", field.Name)
	}
	return
}
