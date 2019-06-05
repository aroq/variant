package variant

import (
	"fmt"

	"github.com/mumoshu/variant/pkg/util/maputil"
	"github.com/pkg/errors"
	"log"
	"reflect"
)

type OrStepLoader struct{}

func (l OrStepLoader) LoadStep(config StepDef, context LoadingContext) (Step, error) {
	data := config.Get("or")

	if data == nil {
		return nil, fmt.Errorf("no field named or exists, config=%v", config)
	}

	steps, ok := data.([]interface{})

	if !ok {
		return nil, fmt.Errorf("field \"or\" must be a map but it wasn't: %v", data)
	}

	result := OrStep{
		Name:   config.GetName(),
		Steps:  []Step{},
		Silent: config.GetBool("silent"),
	}

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

		result.Steps = append(result.Steps, step)
	}

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

func NewOrStepLoader() OrStepLoader {
	return OrStepLoader{}
}

type OrStep struct {
	Name           string
	Steps          []Step
	Silent         bool
	loggingOptions LoggingOptions
}

func (s OrStep) Run(context ExecutionContext) (StepStringOutput, error) {
	var lastError error
	for _, s := range s.Steps {
		var output StepStringOutput

		output, lastError = s.Run(context)

		if lastError == nil {
			return output, nil
		}
	}
	return StepStringOutput{String: "all steps failed"}, errors.Wrapf(lastError, "all steps failed")
}

func (s OrStep) GetName() string {
	return s.Name
}

func (s OrStep) Silenced() bool {
	return s.Silent
}

func (s OrStep) LoggingOpts() LoggingOptions {
	return s.loggingOptions
}
