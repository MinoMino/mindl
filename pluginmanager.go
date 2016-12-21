package main

// mindl - A downloader for various sites and services.
// Copyright (C) 2016  Mino <mino@minomino.org>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/MinoMino/mindl/plugins"
)

type PluginManager []Plugin

var (
	ErrUnintelligibleNumber = errors.New("Unintellible number.")
	ErrOutOfRange           = errors.New("Index out of range.")
	ErrNoPlugins            = errors.New("No plugins to select from.")
	ErrUnsetRequired        = errors.New("A required plugin option was not set and prompting is off.")
	ErrRequiredHidden       = errors.New("A required plugin option is also hidden.")
)

func (pm *PluginManager) FindHandlers(urls []string) [][]Plugin {
	res := make([][]Plugin, len(urls))
	for i, url := range urls {
		handlers := make([]Plugin, 0, 3)
		for _, p := range []Plugin(*pm) {
			if p.CanHandle(url) {
				handlers = append(handlers, p)
			}
		}
		res[i] = handlers
	}

	return res
}

// Returns the plugin if the passed slice only has one plugin,
// otherwise let the user pick the desired plugin.
func (pm *PluginManager) SelectPlugin(ps []Plugin) (Plugin, error) {
	if len(ps) == 0 {
		return nil, ErrNoPlugins
	} else if len(ps) == 1 {
		return ps[0], nil
	}

	fmt.Println("Found multiple handlers. Please select one:")
	for i, p := range ps {
		fmt.Printf("  %2d) %s\n", i+1, p.Name())
	}

	if n, err := strconv.Atoi(prompt("Desired plugin: ")); err != nil {
		return nil, ErrUnintelligibleNumber
	} else if n < 1 || n > len(ps) {
		return nil, ErrOutOfRange
	} else {
		return ps[n-1], nil
	}
}

// Set a plugin's options, prompting the user for missing required fields.
// If prompting isn't desired, return an error instead if required fields
// are unset.
func (pm *PluginManager) SetOptions(ps []Plugin, usropts map[string]string, defaults, noprompt bool) error {
	// A map of all unset options.
	unset := make(map[Plugin][]Option)
	// A map of all unset required options.
	unsetReq := make(map[Plugin][]Option)
	for _, p := range ps {
		plgopts := p.Options()
		for _, plgopt := range plgopts {
			set := false
			for usrkey, usrval := range usropts {
				if strings.EqualFold(plgopt.Key(), usrkey) {
					if err := plgopt.Set(usrval); err != nil {
						return err
					}
					set = true
					log.WithField("plugin", pluginName(p)).Debugf("Set Option: %s = %s",
						plgopt.Key(), usrval)
				}
			}

			// If unset, populate the above maps.
			if !set {
				if plgopt.IsRequired() {
					// An option can't be required and hidden.
					if plgopt.IsHidden() {
						return ErrRequiredHidden
					}
					unsetReq[p] = append(unsetReq[p], plgopt)
				}

				unset[p] = append(unset[p], plgopt)
			}
		}
	}

	if noprompt {
		if len(unsetReq) == 0 { // No prompt, but all required options set?
			return nil
		} else {
			// To make the user aware of which fields weren't set, we log errors.
			for p, opts := range unsetReq {
				for _, opt := range opts {
					log.Errorf("%s: \"%s\" is a required option, but was not set.",
						pluginName(p), opt.Key())
				}
			}
			return ErrUnsetRequired
		}
	} else {
		if defaults {
			// If we're prompting, but defaults is on, only prompt required options.
			for p, opts := range unsetReq {
				name := pluginName(p)
				fmt.Printf("The plugin \"%s\" has required option(s):\n", name)
				for _, opt := range opts {
					// Hidden options are never prompted.
					if opt.IsHidden() {
						continue
					}

					optionPrompt(opt)
					log.WithField("plugin", name).Debugf("Set Option: %s = %s", opt.Key(), opt.Value())
				}
			}
		} else {
			// We're prompting and defaults is off. Prompt everything missing.
			for p, opts := range unset {
				name := pluginName(p)
				fmt.Printf("The plugin \"%s\" has option(s):\n", name)
				for _, opt := range opts {
					// Hidden options are never prompted.
					if opt.IsHidden() {
						continue
					}

					optionPrompt(opt)
					log.WithField("plugin", name).Debugf("Set Option: %s = %s", opt.Key(), opt.Value())
				}
			}
		}
	}

	return nil
}

func prompt(msg string) string {
	r := bufio.NewReader(os.Stdin)
	fmt.Print(msg + ": ")
	in, _ := r.ReadString('\n')

	return strings.TrimSpace(in)
}

func optionPrompt(opt Option) {
	comment := opt.Comment()
	if comment != "" {
		fmt.Println(comment)
	}

	var s, asterisk string
	if opt.IsRequired() {
		asterisk = "*"
	}

	def := fmt.Sprintf("%v", opt.Value()) != "" && !opt.IsRequired()
	if def {
		s = fmt.Sprintf("    %s [%v]%s", opt.Key(), opt.Value(), asterisk)
	} else {
		s = fmt.Sprintf("    %s%s", opt.Key(), asterisk)
	}

	var in string
	for {
		in = prompt(s)
		if in == "" {
			if opt.IsRequired() { // Don't allow empty on required.
				continue
			} else { // Leave default value as is.
				break
			}
		} else if err := opt.Set(in); err != nil {
			log.Error(err)
		} else {
			break
		}
	}
}

func pluginName(p Plugin) string {
	return strings.TrimSpace(p.Name() + " " + p.Version())
}

// Get all special options set by the plugin.
func GetSpecialOptions(p Plugin) map[string]Option {
	res := make(map[string]Option)
	for _, opt := range p.Options() {
		if strings.HasPrefix(opt.Key(), "!") {
			res[opt.Key()[1:]] = opt
		}
	}

	return res
}
