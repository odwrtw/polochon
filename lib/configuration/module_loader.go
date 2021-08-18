package configuration

import (
	"errors"

	polochon "github.com/odwrtw/polochon/lib"
)

// Custom errors
var (
	ErrMissingModuleParams = errors.New("configuration: missing module params")
)

// ModuleLoader is an helper to be embeded in the configuration structure. It
// gets the module names from yaml and loads the module using the modules
// parameters.
type ModuleLoader struct {
	modulesParams *ModulesParams

	TorrenterNames  []string `yaml:"torrenters"`
	DetailerNames   []string `yaml:"detailers"`
	SubtitlerNames  []string `yaml:"subtitlers"`
	SearcherNames   []string `yaml:"searchers"`
	ExplorerNames   []string `yaml:"explorers"`
	NotifierNames   []string `yaml:"notifiers"`
	WishlisterNames []string `yaml:"wishlisters"`
	GuesserNames    []string `yaml:"guessers"`
	CalendarName    string   `yaml:"calendar"`
	FsNotifierName  string   `yaml:"fsnotifier"`
	DownloaderName  string   `yaml:"client"` // TODO: fix the name

	detailers   []polochon.Detailer
	torrenters  []polochon.Torrenter
	subtitlers  []polochon.Subtitler
	explorers   []polochon.Explorer
	searchers   []polochon.Searcher
	notifiers   []polochon.Notifier
	wishlisters []polochon.Wishlister
	guessers    []polochon.Guesser
	calendar    polochon.Calendar
	fsNotifier  polochon.FsNotifier
	downloader  polochon.Downloader
}

func (ml *ModuleLoader) load() error {
	if ml.modulesParams == nil {
		return ErrMissingModuleParams
	}

	var err error
	if len(ml.DetailerNames) != 0 {
		ml.detailers, err = ml.modulesParams.getDetailers(ml.DetailerNames)
		if err != nil {
			return err
		}
	}

	if len(ml.TorrenterNames) != 0 {
		ml.torrenters, err = ml.modulesParams.getTorrenters(ml.TorrenterNames)
		if err != nil {
			return err
		}
	}

	if len(ml.SubtitlerNames) != 0 {
		ml.subtitlers, err = ml.modulesParams.getSubtitlers(ml.SubtitlerNames)
		if err != nil {
			return err
		}
	}

	if len(ml.ExplorerNames) != 0 {
		ml.explorers, err = ml.modulesParams.getExplorers(ml.ExplorerNames)
		if err != nil {
			return err
		}
	}

	if len(ml.SearcherNames) != 0 {
		ml.searchers, err = ml.modulesParams.getSearchers(ml.SearcherNames)
		if err != nil {
			return err
		}
	}

	if len(ml.NotifierNames) != 0 {
		ml.notifiers, err = ml.modulesParams.getNotifiers(ml.NotifierNames)
		if err != nil {
			return err
		}
	}

	if len(ml.WishlisterNames) != 0 {
		ml.wishlisters, err = ml.modulesParams.getWishlisters(ml.WishlisterNames)
		if err != nil {
			return err
		}
	}

	if len(ml.GuesserNames) != 0 {
		ml.guessers, err = ml.modulesParams.getGuessers(ml.GuesserNames)
		if err != nil {
			return err
		}
	}

	if ml.CalendarName != "" {
		ml.calendar, err = ml.modulesParams.getCalendar(ml.CalendarName)
		if err != nil {
			return err
		}
	}

	if ml.FsNotifierName != "" {
		ml.fsNotifier, err = ml.modulesParams.getFsNotifier(ml.FsNotifierName)
		if err != nil {
			return err
		}
	}

	if ml.DownloaderName != "" {
		ml.downloader, err = ml.modulesParams.getDownloader(ml.DownloaderName)
		if err != nil {
			return err
		}
	}

	return nil
}
