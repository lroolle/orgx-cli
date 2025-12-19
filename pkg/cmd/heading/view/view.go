package view

import (
	"fmt"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmd/heading/shared"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/spf13/cobra"
)

type ViewOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Ref    string
	Format string
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "view <ref>",
		Short: "View a heading",
		Long:  "View a specific heading by its reference (path::ID:uuid or path::/Outline).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ref = args[0]

			if runF != nil {
				return runF(opts)
			}
			return viewRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Format, "format", "org", "Output format: org, md, text")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, shared.HeadingFields)

	return cmd
}

func viewRun(opts *ViewOptions) error {
	ref, err := shared.ParseRefFromArg(opts.Ref)
	if err != nil {
		return err
	}

	heading, err := shared.FindHeading(ref)
	if err != nil {
		return err
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, heading)
	}

	return printHeading(opts.IO, heading, opts.Format)
}

func printHeading(io *iostreams.IOStreams, h *ir.Heading, format string) error {
	switch format {
	case "org":
		return printOrgFormat(io, h)
	case "md":
		return printMdFormat(io, h)
	case "text":
		return printTextFormat(io, h)
	default:
		return printOrgFormat(io, h)
	}
}

func printOrgFormat(io *iostreams.IOStreams, h *ir.Heading) error {
	line := strings.Repeat("*", h.Level)
	if h.Todo != "" {
		line += " " + h.Todo
	}
	line += " " + h.Title
	if len(h.Tags) > 0 {
		line += " :" + strings.Join(h.Tags, ":") + ":"
	}
	fmt.Fprintln(io.Out, line)

	if h.Scheduled != "" {
		fmt.Fprintf(io.Out, "SCHEDULED: %s\n", h.Scheduled)
	}
	if h.Deadline != "" {
		fmt.Fprintf(io.Out, "DEADLINE: %s\n", h.Deadline)
	}

	if len(h.Props) > 0 {
		fmt.Fprintln(io.Out, ":PROPERTIES:")
		for k, v := range h.Props {
			fmt.Fprintf(io.Out, ":%s: %s\n", k, v)
		}
		fmt.Fprintln(io.Out, ":END:")
	}

	if h.Body.Raw != "" {
		fmt.Fprintln(io.Out)
		fmt.Fprintln(io.Out, h.Body.Raw)
	}

	return nil
}

func printMdFormat(io *iostreams.IOStreams, h *ir.Heading) error {
	line := strings.Repeat("#", h.Level) + " " + h.Title
	fmt.Fprintln(io.Out, line)

	if h.Todo != "" {
		fmt.Fprintf(io.Out, "**Status:** %s\n", h.Todo)
	}
	if len(h.Tags) > 0 {
		fmt.Fprintf(io.Out, "**Tags:** %s\n", strings.Join(h.Tags, ", "))
	}
	if h.Scheduled != "" {
		fmt.Fprintf(io.Out, "**Scheduled:** %s\n", h.Scheduled)
	}
	if h.Deadline != "" {
		fmt.Fprintf(io.Out, "**Deadline:** %s\n", h.Deadline)
	}

	if h.Body.Raw != "" {
		fmt.Fprintln(io.Out)
		fmt.Fprintln(io.Out, h.Body.Raw)
	}

	return nil
}

func printTextFormat(io *iostreams.IOStreams, h *ir.Heading) error {
	fmt.Fprintf(io.Out, "Title: %s\n", h.Title)
	fmt.Fprintf(io.Out, "Level: %d\n", h.Level)
	fmt.Fprintf(io.Out, "Ref: %s\n", h.Ref)
	if h.Todo != "" {
		fmt.Fprintf(io.Out, "Todo: %s\n", h.Todo)
	}
	if len(h.Tags) > 0 {
		fmt.Fprintf(io.Out, "Tags: %s\n", strings.Join(h.Tags, ", "))
	}
	if h.Scheduled != "" {
		fmt.Fprintf(io.Out, "Scheduled: %s\n", h.Scheduled)
	}
	if h.Deadline != "" {
		fmt.Fprintf(io.Out, "Deadline: %s\n", h.Deadline)
	}
	if h.Body.Raw != "" {
		fmt.Fprintln(io.Out)
		fmt.Fprintln(io.Out, h.Body.Raw)
	}

	return nil
}
