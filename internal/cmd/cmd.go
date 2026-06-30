package cmd

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/spf13/cobra"
	"golang.design/x/clipboard"
	"golang.org/x/text/language"
)

var (
	// go:embed locale.*.toml
	localeFS  embed.FS
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
)

var (
	defaultDir   string
	matchPattern string
	runOnce      bool
)

var command = &cobra.Command{
	Use:  "handheld-path-copier [folder]",
	Args: cobra.RangeArgs(0, 1),
	Run:  run,
}

func init() {
	locales, _ := locale.GetLocales()
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.LoadMessageFileFS(localeFS, "locale.zh.toml")
	localizer = i18n.NewLocalizer(bundle, locales...)

	cobra.MousetrapHelpText = ""

	defaultDir, _ = os.Getwd()
	if homeDir, err := os.UserHomeDir(); err == nil {
		downloadsFolder := filepath.Join(homeDir, "Downloads")
		if info, err := os.Stat(downloadsFolder); err == nil && info.IsDir() {
			defaultDir = downloadsFolder
		}
	}

	command.Flags().StringVar(&matchPattern, "match", "*.dat", "pattern to match the files")
	command.Flags().BoolVar(&runOnce, "once", false, "copy the path once, then exit the program")
}

func Execute(version string) {
	command.Version = version

	if err := command.Execute(); err != nil {
		fatal(err)
	}
}

func run(cmd *cobra.Command, args []string) {
	fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "title",
			Other: "Handheld Data Path Copier version {{.Version}} by Jason Kwok",
		},
		TemplateData: cmd,
	}))
	fmt.Println()

	if err := clipboard.Init(); err != nil {
		fatal(err)
	}

	dir := defaultDir
	if len(args) == 1 {
		path, err := filepath.Abs(args[0])
		if err != nil {
			fatal(err)
		}

		info, err := os.Stat(path)
		if err != nil {
			switch {
			case errors.Is(err, os.ErrNotExist):
				fatal(localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "error_path_not_exist",
						Other: "The path \"{{.Path}}\" does not exist!",
					},
					TemplateData: map[string]string{"Path": path},
				}))
			default:
				fatal(err)
			}
		}

		if !info.IsDir() {
			fatal(localizer.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "error_path_not_folder",
					Other: "The path \"{{.Path}}\" is not a folder!",
				},
				TemplateData: map[string]string{"Path": path},
			}))
		}

		dir = path
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
					if matched, _ := filepath.Match(matchPattern, filepath.Base(event.Name)); matched {
						fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
							DefaultMessage: &i18n.Message{
								ID:    "message_copied",
								Other: "Copied \"{{.Name}}\" to clipboard.",
							},
							TemplateData: event,
						}))
						clipboard.Write(clipboard.FmtText, []byte(event.Name))

						if runOnce {
							stop()
							return
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fatal(err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		fatal(err)
	}
	fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "message_watching",
			Other: "Started watching \"{{.Dir}}\" for new handheld data. Press Ctrl-C to stop.",
		},
		TemplateData: map[string]string{"Dir": dir},
	}))

	<-ctx.Done()
	if err := watcher.Close(); err != nil {
		fatal(err)
	}
}

func fatal(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}
