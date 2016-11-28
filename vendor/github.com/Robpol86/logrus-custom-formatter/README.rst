=======================
logrus-custom-formatter
=======================

Customizable Logrus formatter similar in style to Python's
`logging.Formatter <https://docs.python.org/3.6/library/logging.html#logrecord-attributes>`_.

* Tested with Golang 1.7 on Linux, OS X, and Windows.

📖 Full documentation: https://godoc.org/github.com/Robpol86/logrus-custom-formatter

.. image:: https://img.shields.io/appveyor/ci/Robpol86/logrus-custom-formatter/master.svg?style=flat-square&label=AppVeyor%20CI
    :target: https://ci.appveyor.com/project/Robpol86/logrus-custom-formatter
    :alt: Build Status Windows

.. image:: https://img.shields.io/travis/Robpol86/logrus-custom-formatter/master.svg?style=flat-square&label=Travis%20CI
    :target: https://travis-ci.org/Robpol86/logrus-custom-formatter
    :alt: Build Status

.. image:: https://img.shields.io/codecov/c/github/Robpol86/logrus-custom-formatter/master.svg?style=flat-square&label=Codecov
    :target: https://codecov.io/gh/Robpol86/logrus-custom-formatter
    :alt: Coverage Status

Example
=======

.. image:: examples.png?raw=true
   :alt: Example from Documentation

Quickstart
==========

Install:

.. code:: bash

    go get github.com/Robpol86/logrus-custom-formatter

Usage:

.. code:: go

    // import lcf "github.com/Robpol86/logrus-custom-formatter"
    // import "github.com/Sirupsen/logrus"
    lcf.WindowsEnableNativeANSI(true)
    template := "%[shortLevelName]s[%04[relativeCreated]d] %-45[message]s%[fields]s\n"
    logrus.SetFormatter(lcf.NewFormatter(template, nil))

.. changelog-section-start

Changelog
=========

This project adheres to `Semantic Versioning <http://semver.org/>`_.

1.0.1 - 2016-11-14
------------------

Fixed
    * Newline characters in Basic, Message, and Detailed templates.
    * String padding alignment with ANSI color text (log level names).
    * https://github.com/Robpol86/logrus-custom-formatter/issues/2

1.0.0 - 2016-11-06
------------------

* Initial release.

.. changelog-section-end
