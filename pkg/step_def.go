package variant

import (
	"github.com/mumoshu/variant/pkg/util/maputil"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"reflect"
)

type StepDef struct {
	raw map[string]interface{}
}

func (c StepDef) GetName() string {
	str, ok := c.raw["name"].(string)

	if !ok {
		logrus.Panicf("name wasn't string! name=%s raw=%v", reflect.TypeOf(c.raw["name"]), c.raw)
	}

	return str
}

func (c StepDef) Script() (string, bool) {
	r, ok := c.Get("script").(string)
	return r, ok
}

func (c StepDef) Raw() map[string]interface{} {
	return c.raw
}

func (c StepDef) Get(key string) interface{} {
	return c.raw[key]
}

func (c StepDef) GetStringMapOrEmpty(key string) map[string]interface{} {
	ctx := log.WithField("raw", c.raw[key]).WithField("key", key).WithField("type", reflect.TypeOf(c.raw[key]))

	rawMap, expected := c.Get(key).(map[interface{}]interface{})

	if !expected {
		ctx.Debugf("step config ignored field with unepected type")
		return map[string]interface{}{}
	} else {
		result, err := maputil.CastKeysToStrings(rawMap)

		if err != nil {
			ctx.WithField("error", err).Debugf("step config failed to cast keys to strings")
			return map[string]interface{}{}
		}

		return result
	}
}

func (c StepDef) GetBool(key string) bool {
	result, ok := c.raw[key].(bool)
	if !ok {
		result = false
	}
	return result
}

func NewStepDef(raw map[string]interface{}) StepDef {
	return StepDef{
		raw: raw,
	}
}
