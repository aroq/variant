package variant

import (
	"errors"
	"fmt"
	"github.com/mitchellh/colorstring"
	"github.com/sirupsen/logrus"
	"strings"
)

type variantTextFormatter struct {
	colorize *colorstring.Colorize
	colors   map[logrus.Level]string
	prefixes map[string]string
}

func (f *variantTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var p = "[" + f.colors[entry.Level] + "]"
	var prefix= "%s"
	var prefixApp = "%s%s ≫ "
	var prefixAppTask = "%s%s.%s ≫ "
	if step, ok := entry.Data["step"].(Step); ok {
		prefix = step.LoggingOpts().LogMessagePrefix
		prefixApp = step.LoggingOpts().LogMessagePrefixApp
		prefixAppTask = step.LoggingOpts().LogMessagePrefixAppTask
	}
	app := entry.Data["app"]
	if app != nil {
		switch app := app.(type) {
		case string:
			task := entry.Data["task"]
			if task != nil {
				switch task := task.(type) {
				case string:
					p, _ = TruncatingSprintf(prefixAppTask, p, app, task)
				}
			} else {
				p, _ = TruncatingSprintf(prefixApp, p, app)
			}
		}
	} else {
		p, _ = TruncatingSprintf(prefix, p)
	}
	return []byte(f.colorize.Color(fmt.Sprintf("%s%s\n", p, entry.Message))), nil
}

func TruncatingSprintf(str string, args ...interface{}) (string, error) {
    n := strings.Count(str, "%s")
    if n > len(args) {
        return "", errors.New("Unexpected string:" + str)
    }
    return fmt.Sprintf(str, args[:n]...), nil
}