# mindl
A downloader for various sites and services.

It was written primarily for the purpose of downloading e-books from sites that use HTML5 readers, which is
why some plugins require PhantomJS, but it is not limited to that.

If you've got some other HTML5 reader you want supported and you can provide a sample, I will consider writing a plugin for it.
Open an issue here or send me an e-mail about it. I cannot promise a plugin that downloads from their API if the images are
scrambled because reverse engineering heavily obfuscated JavaScript can be hard and very time consuming, but the PhantomJS approach is
usually fairly easy in contrast.

**See the [releases](https://github.com/MinoMino/mindl/releases/latest) for binaries. For plugins that require PhantomJS, get it [here](http://phantomjs.org/download.html) and place the executable in your working directory or in the PATH environment variable.**

## Building
Linux:
```
go get github.com/MinoMino/mindl
cd $GOPATH/src/github.com/MinoMino/mindl
make
```

Windows (with MSYS binaries in PATH):
```
go get github.com/MinoMino/mindl
cd %GOPATH%\src\github.com\MinoMino\mindl
make
```

## Usage
```
Usage of mindl:
  -d, --defaults             Set to use default values for options whenever possible. No effect if --no-prompt is on.
  -D, --directory string     The directory in which to save the downloaded files. (default "downloads/")
  -n, --no-prompt            Set to turn off prompts for options and instead throw an error if a required option is left unset.
  -o, --option key=value     Options in a key=value format passed to plugins.
  -v, --verbose              Set to display debug messages.
  -w, --workers int          The number of workers to use. (default 10)
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

# License
mindl is licensed under AGPLv3. Refer to `LICENSE` for details.

Refer to the respective licenses and `THIRD-PARTY-NOTICES` for the code under the `vendor` directory
and third-party libraries used.
