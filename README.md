# mindl
A downloader for various sites and services.

It was written primarily for the purpose of downloading e-books from sites that use HTML5 readers, which is
why some plugins require PhantomJS, but it is not limited to that.

If you've got some other HTML5 reader you want supported and you can provide a sample, I will consider writing a plugin for it.
Open an issue here or send me an e-mail about it. I cannot promise a plugin that downloads from their API if the images are
scrambled because reverse engineering heavily obfuscated JavaScript can be hard and very time consuming, but the PhantomJS approach is
usually fairly easy in contrast.

## Installation
The easiest way is to simply use the precompiled binaries:
  1. Get the archive that corresponds to your OS [here](https://github.com/MinoMino/mindl/releases/latest). Windows, Linux and Mac is supported.
  2. Extract the archive wherever you want.
  3. If a plugin you're using needs PhantomJS (e.g. eBookJapan), get it [here](http://phantomjs.org/download.html) and put the executable in the same directory as mindl. The executable is called `phantomjs.exe` on Windows and just `phantomjs` for the others.
  4. Just run mindl from the command line and you're good to go!

If you want to be able to run mindl from anywhere and not just the directory with the executable, just add said directory to your
PATH environment variable.

## Building
This section is only for those who wish to build mindl from source.

Make sure you have [Go](https://golang.org/dl/) installed first, then follow the below instructions:

Linux:
```
go get github.com/MinoMino/mindl
cd $GOPATH/src/github.com/MinoMino/mindl
make
```

Windows (with [MSYS](http://www.mingw.org/wiki/MSYS) binaries in PATH):
```
go get github.com/MinoMino/mindl
cd %GOPATH%\src\github.com\MinoMino\mindl
make
```

## Usage
```
Usage of mindl:
  -d, --defaults           Set to use default values for options whenever possible. No effect if --no-prompt is on.
  -D, --directory string   The directory in which to save the downloaded files. (default "downloads/")
  -n, --no-prompt          Set to turn off prompts for options and instead throw an error if a required option is left unset.
  -o, --option key=value   Options in a key=value format passed to plugins.
  -v, --verbose            Set to display debug messages.
  -w, --workers int        The number of workers to use. (default 10)
  -z, --zip                Set to ZIP the files after the download finishes.
```

### Example
```
mino$ mindl -d -z -o username=my@email.com -o password=password123 https://bookwalker.jp/de[...]
(00:18:05 INFO) Starting download using "BookWalker"...
(00:18:05 INFO) [BookWalker] Logging in...
(00:18:22 INFO) Zipping files to: SomeBook.zip
(00:18:24 INFO) Cleaning up...
(00:18:24 INFO) [BookWalker] Logging out...
(00:18:27 INFO) Done! Got a total of 181 downloads.
```

**Make sure you use double quotes around each URL, or the console will interpret the ampersands as multiple console commands
instead of part of the URL(s).**

If the plugin requires any options to be configured, you can pass them with `-o` like in the above example, but you can
also just run mindl without passing them and have it prompt you for them later.

## Supported Services
* [eBookJapan](https://github.com/MinoMino/mindl/wiki/Supported-Services#ebookjapan)
* [BookLive](https://github.com/MinoMino/mindl/wiki/Supported-Services#booklive)
* [BookWalker](https://github.com/MinoMino/mindl/wiki/Supported-Services#bookwalker) (**read before using**)

# License
mindl is licensed under AGPLv3. Refer to `LICENSE` for details.

Refer to the respective licenses and `THIRD-PARTY-NOTICES` for the code under the `vendor` directory
and third-party libraries used.
