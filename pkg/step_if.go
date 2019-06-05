package variant

import (
	"fmt"

	"github.com/mumoshu/variant/pkg/util/maputil"
	"github.com/pkg/errors"
	"log"
	"reflect"
)

type IfStepLoader struct{}

func (l IfStepLoader) LoadStep(config StepDef, context LoadingContext) (Step, error) {
	ifData := config.Get("if")

	if ifData == nil {
		return nil, fmt.Errorf("no field named `if` exists, config=%v", config)
	}

	ifArray, ok := ifData.(interface{})

	if !ok {
		return nil, fmt.Errorf("field \"if\" must be an interface{} but it wasn't: %v", ifData)
	}

	thenData := config.Get("then")

	if thenData == nil {
		return nil, fmt.Errorf("no field named `then` exists, config=%v", config)
	}

	thenArray, ok2 := thenData.(interface{})

	if !ok2 {
		return nil, fmt.Errorf("field \"then\" must be an interface{} but it wasn't: %v", ifData)
	}

	result := IfStep{
		Name:   config.GetName(),
		If:     []Step{},
		Then:   []Step{},
		Silent: config.GetBool("silent"),
	}

	ifInput, ifErr := readSteps(ifArray, context)

	if ifErr != nil {
		return nil, errors.Wrapf(ifErr, "reading `if` failed")
	}

	thenInput, thenErr := readSteps(thenArray, context)

	if thenErr != nil {
		return nil, errors.Wrapf(thenErr, "reading `then` failed")
	}

	result.If = ifInput
	result.Then = thenInput

	var loggingOptions *LoggingOptions
	outputOptionsContainer, ok := config.Get("logging").(map[interface{}]interface{})
	if ok {
		loggingOptions = &LoggingOptions{
			SuppressStdOut:          false,
			SuppressStdErr:          false,
			RedirectStdErrToStdOut:  false,
			ExitErrorLogLevel:       "error",
			LogMessagePrefixApp:     "%s%s ≫ ",
			LogMessagePrefixAppTask: "%s%s.%s ≫ ",
		}
		if v, ok := outputOptionsContainer["suppress_stdout"].(bool); ok {
			loggingOptions.SuppressStdOut = v
		}
		if v, ok := outputOptionsContainer["suppress_stderr"].(bool); ok {
			loggingOptions.SuppressStdErr = v
		}
		if v, ok := outputOptionsContainer["exit_error_log_level"].(string); ok {
			loggingOptions.ExitErrorLogLevel = v
		}
		if v, ok := outputOptionsContainer["redirect_stderr_to_stdout"].(bool); ok {
			loggingOptions.RedirectStdErrToStdOut = v
		}
		if v, ok := outputOptionsContainer["log_message_prefix"].(string); ok {
			loggingOptions.LogMessagePrefix = v
		}
		if v, ok := outputOptionsContainer["log_message_prefix_app"].(string); ok {
			loggingOptions.LogMessagePrefixApp = v
		}
		if v, ok := outputOptionsContainer["log_message_prefix_app_task"].(string); ok {
			loggingOptions.LogMessagePrefixAppTask = v
		}

		result.loggingOptions = *loggingOptions
	}

	return result, nil
}

func readSteps(input interface{}, context LoadingContext) ([]Step, error) {
	steps, ok := input.([]interface{})

	if !ok {
		return nil, fmt.Errorf("input must be array: input=%v", input)
	}

	result := []Step{}

	for i, s := range steps {
		stepAsMap, isMap := s.(map[interface{}]interface{})

		if !isMap {
			log.Panicf("isnt map! %s", reflect.TypeOf(s))
		}

		converted, conversionErr := maputil.CastKeysToStrings(stepAsMap)
		if conversionErr != nil {
			return nil, errors.WithStack(conversionErr)
		}

		if converted["name"] == "" || converted["name"] == nil {
			converted["name"] = fmt.Sprintf("or[%d]", i)
		}

		step, loadingErr := context.LoadStep(NewStepDef(converted))
		if loadingErr != nil {
			return nil, errors.WithStack(loadingErr)
		}

		result = append(result, step)
	}

	return result, nil
}

func NewIfStepLoader() IfStepLoader {
	return IfStepLoader{}
}

type IfStep struct {
	Name           string
	If             []Step
	Then           []Step
	Silent         bool
	loggingOptions LoggingOptions
}

func run(steps []Step, context ExecutionContext) (StepStringOutput, error) {
	var lastOutput StepStringOutput
	var lastError error

	for _, s := range steps {
		lastOutput, lastError = s.Run(context)

		if lastError != nil {
			return StepStringOutput{String: "run error"}, errors.Wrapf(lastError, "failed running step")
		}
	}

	return lastOutput, nil
}

func (s IfStep) Run(context ExecutionContext) (StepStringOutput, error) {
	_, ifErr := run(s.If, context)

	if ifErr != nil {
		return StepStringOutput{String: "if step failed"}, nil
	}

	thenOut, thenErr := run(s.Then, context)

	if thenErr != nil {
		return StepStringOutput{String: "then step failed"}, errors.Wrapf(thenErr, "`then` steps failed")
	}

	return thenOut, nil
}

func (s IfStep) GetName() string {
	return s.Name
}

func (s IfStep) Silenced() bool {
	return s.Silent
}

func (s IfStep) LoggingOpts() LoggingOptions {
	return s.loggingOptions
}
