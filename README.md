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
  3. If a plugin you're using needs PhantomJS (e.g. EBookJapan), get it [here](http://phantomjs.org/download.html) and put the executable in the same directory as mindl. The executable is called `phantomjs.exe` on Windows and just `phantomjs` for the others.
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
mino$ mindl -d -o jpegquality=80 "https://br.ebookjapan.jp/br/reader/viewer/view.html?sessionid=[...]"
INFO[0000] Starting download using "EBookJapan"...
INFO[0000] Starting PhantomJS...
INFO[0001] Opening the reader...
INFO[0006] Waiting for reader to load...
INFO[0133] Done! Got a total of 76 downloads.
```

**Make sure you use double quotes around each URL, or the console will interpret the ampersands as multiple console commands
instead of part of the URL(s).**

If the plugin requires any options to be configured, you can pass them with `-o` like in the above example, but you can
also just run mindl without passing them and have it prompt you for them later.

## Supported Sites
### eBookJapan
Uses PhantomJS to open the reader and download the pages. This approach is somewhat slow and CPU/RAM heavy, but
they've done a good job at making it a pain in the ass to do it any other way.

Note that since everything is bottlenecked through one instance of PhantomJS, the `--worker` flag doesn't do anything.
Instead you can adjust the `PrefetchCount` option for some concurrency on the PhantomJS side. `PrefetchCount` has
diminishing returns, so don't go too crazy.

##### Usage
Open the reader and you should get a URL like
`https://br.ebookjapan.jp/br/reader/viewer/view.html?sessionid=[...]&keydata=[...]&shopID=eBookJapan`
which is the one you need to pass to mindl. EBJ has protection against account sharing, so make sure
you both get the URL *and* use mindl from the same IP address.

### BookLive
Directly interacts with the API and descrambles the images concurrently, so it's very fast and efficient.
If you do not own the book, it will instead download the trial pages.

##### Usage
The URLs handled by this plugin:
* Product pages: `https://booklive.jp/product/index/title_id/[...]/vol_no/[...]`
* Reader: `https://booklive.jp/bviewer/?cid=[...]&rurl=[...]`

### BookWalker
Also directly interacts with the API, so it's very fast. If you've recently opened a book through
your browser, log out of your account first. This is because BookWalker prevents you from opening
books from two different browsers.

**IMPORTANT**: Be very careful with how much you download. They seem to have some sort of threshold
on the rate at which you download pages from their servers, and if you pass it, they can ban you.
I do not know exactly what this threshold is, but you might want to run it with `--workers 1` in
order to significantly slow down the downloading.

##### Usage
The URLs handled by this plugin:
* Product pages: `https://bookwalker.jp/de[...]`
  * Example: `https://bookwalker.jp/de476913de-9a40-4544-a759-10e59c4c3ec0/カカフカカ-1/`

# License
mindl is licensed under AGPLv3. Refer to `LICENSE` for details.

Refer to the respective licenses and `THIRD-PARTY-NOTICES` for the code under the `vendor` directory
and third-party libraries used.
