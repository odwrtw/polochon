## Modules

Modules needs to be loaded using the "\_" prefix. See [this link](https://golang.org/doc/effective_go.html#blank_import) for more explanation.

Modules are configured in the "modules" section of the configuration. Each module can have specific key / value configuration options.

### Detailer

The detailer module is used to get details for a movie, a show or a show episode. It fills the video type with all the required informations to create a proper NFO file for xmbc.

Available detailers:
* tvdb (shows)
* tmdb (movies)

### Torrenter

The torrenter works with the detailer, it searches for torrents on different sources from the video informations.

Available torrenters:
* yts (movies)
* eztv (shows)

### Guesser

The guesser tries to guess informations from a video file, using regexp or file hash. It returns a proper video type so it can be updated by the detailer and stored into the video library.

Available guesser:
* openguessit (mix of opensubtitle and guessit)

### FsNotifier

The fs notifier notifies the app whenever a file has been added into the watched directory.

Available FsNotifier:
* inotify (linux file system notifier)
* fsnotify (multi-platform file system notifier)

### Notifier

The notifier notifies when a video has been properly stored in the library.

Available Notifier:
* pushover

### Subtitler

The subtitler downloads subtitles and stores them with the video file, so that the video player can open them during playback.

Available Sutitler:
* addicted (shows)
* opensubtitles (movies and shows)
* yifysubtitle (shows)

### Wishlister

The wishlister gets the user(s) wishlists on various websites / services.

Available Wishlister:
* imdb (movies and shows)
* canape (movies and shows)

### Downloader

The downloader sends remote command to a BiTorrent server to add, remove or list entries.

Available Downloader:
* transmission
