package get

import (
	"fmt"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/cmd/heading/shared"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/spf13/cobra"
)

type GetOptions struct {
	IO       *iostreams.IOStreams
	Exporter cmdutil.Exporter

	Ref        string
	NoChildren bool
	Format     string
}

func NewCmdGet(f *cmdutil.Factory, runF func(*GetOptions) error) *cobra.Command {
	opts := &GetOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "get <ref>",
		Short: "Get specific section by ref",
		Long: `Get a specific heading and its content by reference.

Examples:
  orgx get notes.org::ID:abc123
  orgx get notes.org::/Projects/CLI --format md
  orgx get notes.org::ID:abc123 --no-children --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ref = args[0]

			if runF != nil {
				return runF(opts)
			}
			return getRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.NoChildren, "no-children", false, "Heading only, skip children")
	cmd.Flags().StringVar(&opts.Format, "format", "org", "Output format: org, md, text")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, shared.HeadingFields)

	return cmd
}

func getRun(opts *GetOptions) error {
	ref, err := shared.ParseRefFromArg(opts.Ref)
	if err != nil {
		return err
	}

	heading, err := shared.FindHeading(ref)
	if err != nil {
		return err
	}

	if opts.NoChildren {
		heading.Children = nil
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, heading)
	}

	return printHeading(opts.IO, heading, opts.Format)
}

func printHeading(io *iostreams.IOStreams, h *ir.Heading, format string) error {
	switch format {
	case "md":
		return printMd(io, h)
	case "text":
		return printText(io, h)
	default:
		return printOrg(io, h)
	}
}

func printOrg(io *iostreams.IOStreams, h *ir.Heading) error {
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

	for _, child := range h.Children {
		if ch, ok := child.(*ir.Heading); ok {
			fmt.Fprintln(io.Out)
			printOrg(io, ch)
		}
	}

	return nil
}

func printMd(io *iostreams.IOStreams, h *ir.Heading) error {
	fmt.Fprintf(io.Out, "%s %s\n", strings.Repeat("#", h.Level), h.Title)

	if h.Todo != "" || len(h.Tags) > 0 {
		fmt.Fprint(io.Out, "**")
		if h.Todo != "" {
			fmt.Fprintf(io.Out, "%s", h.Todo)
		}
		if len(h.Tags) > 0 {
			if h.Todo != "" {
				fmt.Fprint(io.Out, " ")
			}
			fmt.Fprintf(io.Out, "[%s]", strings.Join(h.Tags, ", "))
		}
		fmt.Fprintln(io.Out, "**")
	}

	if h.Body.Raw != "" {
		fmt.Fprintln(io.Out)
		fmt.Fprintln(io.Out, h.Body.Raw)
	}

	for _, child := range h.Children {
		if ch, ok := child.(*ir.Heading); ok {
			fmt.Fprintln(io.Out)
			printMd(io, ch)
		}
	}

	return nil
}

func printText(io *iostreams.IOStreams, h *ir.Heading) error {
	fmt.Fprintf(io.Out, "Title: %s\n", h.Title)
	fmt.Fprintf(io.Out, "Level: %d\n", h.Level)
	fmt.Fprintf(io.Out, "Ref: %s\n", h.Ref)
	if h.Todo != "" {
		fmt.Fprintf(io.Out, "Todo: %s\n", h.Todo)
	}
	if len(h.Tags) > 0 {
		fmt.Fprintf(io.Out, "Tags: %s\n", strings.Join(h.Tags, ", "))
	}
	if h.Body.Raw != "" {
		fmt.Fprintln(io.Out)
		fmt.Fprintln(io.Out, h.Body.Raw)
	}
	return nil
}
