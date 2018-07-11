package main

import (
	"fmt"

	"github.com/blinkist/skipper/aws/ecsclient"
	"github.com/blinkist/skipper/helpers"
	cwlogs "github.com/segmentio/cwlogs/lib"

	"os"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	verboseFormatString = `[ {{ uniquecolor (print .TaskShort) }} ] {{ .TimeShort }} {{ colorlevel .Level }} {{- range $key, $value := .DataFlat }} {{ printf "%v=%v" $key $value }} {{end}} {{- if gt (len .Info.Errors) 0 }} Errors=[{{- range $value := .Info.Errors }} Type={{ printf "%s" $value.Type }} Error={{ printf "%s" $value.Error }} {{ if $value.Stack }} Stack={{printf "%v" $value.Stack}} {{- end }}{{- end }}] {{ end }} - {{ .Message }}`
	//defaultFormatString = `[ {{ uniquecolor (print .TaskShort) }} ] {{ .TimeShort }} {{ colorlevel .Level }} - {{ .Message }}`
	defaultFormatString = `[ {{ colorlevel .Level }} - {{ .Message }}`
	rawFormatString     = `{{ .PrettyPrint }}`
)

var templateFuncMap = template.FuncMap{
	"red":         cwlogs.Red,
	"green":       cwlogs.Green,
	"yellow":      cwlogs.Yellow,
	"blue":        cwlogs.Blue,
	"magenta":     cwlogs.Magenta,
	"cyan":        cwlogs.Cyan,
	"white":       cwlogs.White,
	"uniquecolor": cwlogs.Unique,
	"colorlevel":  cwlogs.ColorLevel,
}

var (
	follow        bool
	task          string
	eventTemplate string
	since         string
	until         string
	verbose       bool
	raw           bool
)

// Error messages
var (
	ErrTooFewArguments  = errors.New("Too few arguments")
	ErrTooManyArguments = errors.New("Too many arguments")
	ErrNoEventsFound    = errors.New("No log events found")
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "logs [cluster] [service]",
	Short: "fetch logs for a given service",
	RunE:  fetch,
}

func init() {
	RootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringVarP(&task, "task", "t", "", "Task UUID or prefix")
	fetchCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log streams")
	fetchCmd.Flags().StringVarP(&eventTemplate, "format", "o", defaultFormatString, "Format template for displaying log events")
	fetchCmd.Flags().StringVarP(&since, "since", "s", "1h", "Fetch logs since timestamp (e.g. 2013-01-02T13:23:37), relative (e.g. 42m for 42 minutes), or all for all logs")
	fetchCmd.Flags().StringVarP(&until, "until", "u", "now", "Fetch logs until timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes)")
	fetchCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose log output (includes log context in data fields)")
	fetchCmd.Flags().BoolVarP(&raw, "raw", "r", false, "Raw JSON output")
}

func fetch(cmd *cobra.Command, args []string) error {

	start, err := cwlogs.GetTime(since, time.Now())
	if err != nil {
		return fmt.Errorf("failed to parse time '%s'", since)
	}

	var end time.Time
	if cmd.Flags().Lookup("until").Changed {
		if cmd.Flags().Lookup("follow").Changed {
			return fmt.Errorf("can't set both --until and --follow")
		}
		end, err = cwlogs.GetTime(until, time.Now())
		if err != nil {
			return fmt.Errorf("failed to parse time '%s'", until)
		}
	}

	ecs := ecsclient.New()
	cluster, service := helpers.ServicePicker(ecs, args)

	tcs, err := ecs.GetContainerInstances(&cluster, &service)
	if err != nil {
		fmt.Errorf("error getting container instances: %s", err)
		os.Exit(1)
	}

	loggroup := "undefined"
	for _, ti := range tcs {
		loggroup = *ti.AwsLogGroup
	}

	logReader, err := cwlogs.NewCloudwatchLogsReader(loggroup, task, start, end)

	if err != nil {
		return err
	}

	if cmd.Flags().Lookup("verbose").Changed && cmd.Flags().Lookup("raw").Changed {
		return fmt.Errorf("can't set both --raw and --verbose")
	}

	if verbose {
		eventTemplate = verboseFormatString
	}

	if raw {
		eventTemplate = rawFormatString
	}

	output, err := template.New("event").Funcs(templateFuncMap).Parse(eventTemplate)
	if err != nil {
		return err
	}

	eventChan := logReader.StreamEvents(follow)

	ticker := time.After(7 * time.Second)

ReadLoop:
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				break ReadLoop
			}
			err = output.Execute(os.Stdout, event)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "\n")
			// reset slow log warning timer
			ticker = time.After(7 * time.Second)
		case <-ticker:
			if !follow {
				fmt.Fprintf(os.Stdout, "logs are taking a while to load... possibly try a smaller time window")
			}
		}
	}

	return logReader.Error()
}
