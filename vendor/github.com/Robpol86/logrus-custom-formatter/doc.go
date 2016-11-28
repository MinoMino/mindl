/*
Package lcf (logrus-custom-formatter) is a customizable formatter for https://github.com/Sirupsen/logrus that lets you
choose which columns to include in your log outputs.

Windows Support

Unlike Linux/OS X, Windows kind of doesn't support ANSI color codes. Windows versions before Windows 10 Insider Edition
around May 2016 do not support ANSI color codes (instead your program is supposed to issue SetConsoleTextAttribute win32
API calls before each character to display if its color changes) and lcf will disable colors on those platforms by
default. Windows version after that do actually support ANSI color codes but is disabled by default. lcf will detect
this and disable colors by default if this feature (ENABLE_VIRTUAL_TERMINAL_PROCESSING) is not enabled.

You can enable ENABLE_VIRTUAL_TERMINAL_PROCESSING by calling lcf.WindowsEnableNativeANSI(true) in your program (logrus
by default only outputs to stderr, call with false if you're printing to stdout instead). More information in the
WindowsEnableNativeANSI documentation below.

Example Program

Below is a simple example program that uses lcf with logrus:

	package main

	import (
		lcf "github.com/Robpol86/logrus-custom-formatter"
		"github.com/Sirupsen/logrus"
	)

	func main() {
		lcf.WindowsEnableNativeANSI(true)
		temp := "%[shortLevelName]s[%04[relativeCreated]d] %-45[message]s%[fields]s\n"
		logrus.SetFormatter(lcf.NewFormatter(temp, nil))
		logrus.SetLevel(logrus.DebugLevel)

		animal := logrus.Fields{"animal": "walrus", "size": 10}
		logrus.WithFields(animal).Debug("A group of walrus emerges from the ocean")
		logrus.WithFields(animal).Warn("The group's number increased tremendously!")
		number := logrus.Fields{"number": 122, "omg": true}
		logrus.WithFields(number).Info("A giant walrus appears!")
		logrus.Error("Tremendously sized cow enters the ocean.")
	}

And the output is:

	DEBU[0000] A group of walrus emerges from the ocean      animal=walrus size=10
	WARN[0000] The group's number increased tremendously!    animal=walrus size=10
	INFO[0000] A giant walrus appears!                       number=122 omg=true
	ERRO[0000] Tremendously sized cow enters the ocean.

Built-In Attributes

These attributes are provided by lcf and can be specified in your template string:

	%[ascTime]s		Timestamp formatted by CustomFormatter.TimestampFormat.
	%[fields]s		Logrus fields formatted as "key1=value key2=value". Keys are
				sorted unless CustomFormatter.DisableSorting is true.
	%[levelName]s		The capitalized log level name (e.g. INFO, WARNING, ERROR).
	%[message]s		The log message.
	%[name]s		The value of the "name" field. If used "name" will be omitted
				from %[fields]s.
	%[process]d		The current PID of the process emitting log statements.
	%[relativeCreated]d	Number of seconds since the program has started (since
				formatter was created)
	%[shortLevelName]s	Like %[levelName]s except WARNING is shown as "WARN".

Custom Handlers

If what you're looking for is not available in the above built-in attributes or not exactly the functionality that you
want you can add new or override existing attributes with custom handlers. Read the documentation for the CustomHandlers
type below for more information.
*/
package lcf
