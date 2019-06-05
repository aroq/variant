package variant

import (
	"fmt"

	"github.com/mumoshu/variant/pkg/api/task"
)

type TaskStepLoader struct{}

func (l TaskStepLoader) LoadStep(stepConfig StepDef, context LoadingContext) (Step, error) {
	if taskKey, isStr := stepConfig.Get("task").(string); isStr && taskKey != "" {
		inputs := task.NewArguments(stepConfig.GetStringMapOrEmpty("inputs"))
		if len(inputs) == 0 {
			inputs = task.NewArguments(stepConfig.GetStringMapOrEmpty("arguments"))
		}

		var loggingOptions *LoggingOptions
		outputOptionsContainer, ok := stepConfig.Get("logging").(map[interface{}]interface{})
		if ok {
			loggingOptions = &LoggingOptions{
				SuppressStdOut:          false,
				SuppressStdErr:          false,
				RedirectStdErrToStdOut:  false,
				ExitErrorLogLevel:       "error",
				LogMessagePrefix:        "%s%s ≫ ",
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
		}

		step := TaskStep{
			Name:          stepConfig.GetName(),
			TaskKeyString: taskKey,
			Arguments:     inputs,
			Silent:        stepConfig.GetBool("silent"),
		}

		if loggingOptions != nil {
			step.loggingOptions = *loggingOptions
		}

		return step, nil
	}

	return nil, fmt.Errorf("couldn't load task step")
}

func NewTaskStepLoader() TaskStepLoader {
	return TaskStepLoader{}
}

type TaskStep struct {
	Name           string
	TaskKeyString  string
	Arguments      task.Arguments
	Silent         bool
	loggingOptions LoggingOptions
}

func (s TaskStep) Run(context ExecutionContext) (StepStringOutput, error) {
	output, err := context.RunAnotherTask(s.TaskKeyString, s.Arguments.TransformStringValues(func(v string) string {
		v2, err := context.Render(v, "argument")
		if err != nil {
			panic(err)
		}
		return v2
	}), context.Vars())
	return StepStringOutput{String: output}, err
}

func (s TaskStep) GetName() string {
	return s.Name
}

func (s TaskStep) Silenced() bool {
	return s.Silent
}

func (s TaskStep) LoggingOpts() LoggingOptions {
	return s.loggingOptions
}
