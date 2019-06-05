package variant

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mumoshu/variant/pkg/get"
	"github.com/mumoshu/variant/pkg/util/maputil"
	"strings"
)

type TaskDef struct {
	Name        string       `yaml:"name,omitempty"`
	Description string       `yaml:"description,omitempty"`
	Inputs      InputConfigs `yaml:"inputs,omitempty"`
	TaskDefs    TaskDefs     `yaml:"tasks,omitempty"`
	Script      string       `yaml:"script,omitempty"`
	Steps       []Step       `yaml:"steps,omitempty"`
	Autoenv     bool         `yaml:"autoenv,omitempty"`
	Autodir     bool         `yaml:"autodir,omitempty"`
	Interactive bool         `yaml:"interactive,omitempty"`
	Private     bool         `yaml:"private,omitempty"`
	Silent      bool         `yaml:"silent,omitempty"`
}

type TaskDefs []*TaskDef

func (d TaskDefs) GoString() string {
	taskDefs := []string{}
	for _, t := range d {
		taskDefs = append(taskDefs, fmt.Sprintf("%#v", t))
	}
	return fmt.Sprintf("variant.TaskDefs{%s}", strings.Join(taskDefs, ", "))
}

type InputConfigs []*InputConfig

func (c InputConfigs) GoString() string {
	inputConfs := []string{}
	for _, t := range c {
		inputConfs = append(inputConfs, fmt.Sprintf("%#v", t))
	}
	return fmt.Sprintf("variant.InputConfigs{%s}", strings.Join(inputConfs, ", "))
}

type TaskDefV1 struct {
	Name        string                        `yaml:"name,omitempty"`
	Description string                        `yaml:"description,omitempty"`
	Inputs      []*InputConfig                `yaml:"inputs,omitempty"`
	Parameters  []*ParameterConfig            `yaml:"parameters,omitempty"`
	Options     []*OptionConfig               `yaml:"options,omitempty"`
	TaskDefs    []*TaskDef                    `yaml:"tasks,omitempty"`
	Runner      map[string]interface{}        `yaml:"runner,omitempty"`
	Script      string                        `yaml:"script,omitempty"`
	StepDefs    []map[interface{}]interface{} `yaml:"steps,omitempty"`
	Autoenv     bool                          `yaml:"autoenv,omitempty"`
	Autodir     bool                          `yaml:"autodir,omitempty"`
	Interactive bool                          `yaml:"interactive,omitempty"`
	Private     bool                          `yaml:"private,omitempty"`
	Silent      bool                          `yaml:"silent,omitempty"`
}

type TaskDefV2 struct {
	Description string                        `yaml:"description,omitempty"`
	Inputs      []*InputConfig                `yaml:"inputs,omitempty"`
	Parameters  []*ParameterConfig            `yaml:"parameters,omitempty"`
	Options     []*OptionConfig               `yaml:"options,omitempty"`
	Import      string                        `yaml:"import,omitempty"`
	TaskDefs    map[string]*TaskDef           `yaml:"tasks,omitempty"`
	Runner      map[string]interface{}        `yaml:"runner,omitempty"`
	Script      interface{}                   `yaml:"script,omitempty"`
	StepDefs    []map[interface{}]interface{} `yaml:"steps,omitempty"`
	Autoenv     bool                          `yaml:"autoenv,omitempty"`
	Autodir     bool                          `yaml:"autodir,omitempty"`
	Interactive bool                          `yaml:"interactive,omitempty"`
	Private     bool                          `yaml:"private,omitempty"`
	Silent      bool                          `yaml:"silent,omitempty"`
}

func (t *TaskDef) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v3 := map[string]interface{}{}
	if err := unmarshal(&v3); err != nil {
		return err
	}

	log.Tracef("Unmarshalling: %v", v3)

	var v2 *TaskDefV2

	log.Tracef("Trying to parse v2 format")
	v2 = &TaskDefV2{
		Autoenv:     false,
		Autodir:     false,
		Interactive: false,
		Inputs:      []*InputConfig{},
		TaskDefs:    map[string]*TaskDef{},
		StepDefs:    []map[interface{}]interface{}{},
		Silent:      false,
	}

	if err := unmarshal(&v2); err != nil {
		return err
	}

	if v2.Import != "" {
		log.Debugf("Importing %s", v2.Import)

		err := get.Unmarshal(v2.Import, &v2)
		if err != nil {
			return err
		}
	}

	var script string
	switch s := v2.Script.(type) {
	case string:
		script = s
	case []interface{}:
		ss := make([]string, len(s))
		for i := range s {
			ss[i] = s[i].(string)
		}
		script = strings.Join(ss, "\n")
	}

	if len(v2.TaskDefs) == 0 && script == "" && len(v2.StepDefs) == 0 {
		return fmt.Errorf("Not v2 format: `tasks`, `script`, `steps` are missing.")
	}

	t.Description = v2.Description
	if len(v2.Inputs) > 0 {
		t.Inputs = v2.Inputs
	} else {
		for i, p := range v2.Parameters {
			c := i
			input := &InputConfig{
				Name:          p.Name,
				Description:   p.Description,
				ArgumentIndex: &c,
				Type:          p.Type,
				Default:       p.Default,
				Remainings:    p.Remainings,
				Properties:    p.Properties,
			}
			t.Inputs = append(t.Inputs, input)
		}
		for _, o := range v2.Options {
			input := &InputConfig{
				Name:        o.Name,
				Description: o.Description,
				Type:        o.Type,
				Default:     o.Default,
				Remainings:  o.Remainings,
				Properties:  o.Properties,
			}
			t.Inputs = append(t.Inputs, input)
		}
	}
	t.TaskDefs = TransformV2FlowConfigMapToArray(v2.TaskDefs)

	t.Script = script
	t.Autoenv = v2.Autoenv
	t.Autodir = v2.Autodir
	t.Interactive = v2.Interactive
	t.Private = v2.Private
	t.Silent = v2.Silent

	steps, err := readStepsFromStepDefs(t, script, v2.Runner, v2.StepDefs)
	if err != nil {
		return errors.Wrapf(err, "Error while reading v2 config")
	}
	t.Steps = steps
	return nil
}

func (t *TaskDef) CopyTo(other *TaskDef) {
	other.Description = t.Description
	other.Inputs = t.Inputs
	other.TaskDefs = t.TaskDefs
	other.Steps = t.Steps
	other.Script = t.Script
	other.Autoenv = t.Autoenv
	other.Autodir = t.Autodir
	other.Interactive = t.Interactive
	other.Private = t.Private
}

func TransformV2FlowConfigMapToArray(v2 map[string]*TaskDef) []*TaskDef {
	result := []*TaskDef{}
	for name, t2 := range v2 {
		t := &TaskDef{}

		t.Name = name
		t2.CopyTo(t)

		result = append(result, t)
	}
	return result
}

var stepLoaders []StepLoader

func Register(stepLoader StepLoader) {
	stepLoaders = append(stepLoaders, stepLoader)
}

func init() {
	stepLoaders = []StepLoader{}
}

type stepLoadingContextImpl struct{}

func (s stepLoadingContextImpl) LoadStep(config StepDef) (Step, error) {
	step, err := LoadStep(config)

	return step, err
}

func LoadStep(config StepDef) (Step, error) {
	var lastError error

	lastError = nil

	context := stepLoadingContextImpl{}
	for _, loader := range stepLoaders {
		var s Step
		s, lastError = loader.LoadStep(config, context)

		log.WithField("step", s).Debugf("step loaded")

		if lastError == nil {
			return s, nil
		}
	}
	return nil, errors.Wrapf(lastError, "all loader failed to load step")
}

func readStepsFromStepDefs(t *TaskDef, script string, runner map[string]interface{}, stepDefs []map[interface{}]interface{}) ([]Step, error) {
	result := []Step{}

	if script != "" {
		if len(stepDefs) > 0 {
			return nil, fmt.Errorf("both script and steps exist.")
		}

		raw := map[string]interface{}{
			"name":   "script",
			"script": script,
			"silent": t.Silent,
		}
		if runner != nil {
			raw["runner"] = runner
		}
		s, err := LoadStep(NewStepDef(raw))

		if err != nil {
			log.Panicf("step failed to load: %v", err)
		}

		result = []Step{s}
	} else {
		for i, stepDef := range stepDefs {
			defaultName := fmt.Sprintf("step-%d", i+1)

			if stepDef["name"] == "" || stepDef["name"] == nil {
				stepDef["name"] = defaultName
			}

			converted, castErr := maputil.CastKeysToStrings(stepDef)

			if castErr != nil {
				panic(castErr)
			}

			s, err := LoadStep(NewStepDef(converted))

			if err != nil {
				return nil, errors.Wrapf(err, "Error reading step[%d]")
			}

			result = append(result, s)
		}
	}

	return result, nil
}
