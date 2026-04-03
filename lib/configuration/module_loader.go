package configuration

import (
	"errors"
	"log/slog"

	polochon "github.com/odwrtw/polochon/lib"
)

// Custom errors
var ErrMissingModuleParams = errors.New("configuration: missing module params")

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

func (ml *ModuleLoader) load(log *slog.Logger) error {
	if ml.modulesParams == nil {
		return ErrMissingModuleParams
	}
	mp := ml.modulesParams
	var err error

	if ml.detailers, err = getModulesAs[polochon.Detailer](mp, polochon.TypeDetailer, log, ml.DetailerNames); err != nil {
		return err
	}
	if ml.torrenters, err = getModulesAs[polochon.Torrenter](mp, polochon.TypeTorrenter, log, ml.TorrenterNames); err != nil {
		return err
	}
	if ml.subtitlers, err = getModulesAs[polochon.Subtitler](mp, polochon.TypeSubtitler, log, ml.SubtitlerNames); err != nil {
		return err
	}
	if ml.explorers, err = getModulesAs[polochon.Explorer](mp, polochon.TypeExplorer, log, ml.ExplorerNames); err != nil {
		return err
	}
	if ml.searchers, err = getModulesAs[polochon.Searcher](mp, polochon.TypeSearcher, log, ml.SearcherNames); err != nil {
		return err
	}
	if ml.notifiers, err = getModulesAs[polochon.Notifier](mp, polochon.TypeNotifier, log, ml.NotifierNames); err != nil {
		return err
	}
	if ml.wishlisters, err = getModulesAs[polochon.Wishlister](mp, polochon.TypeWishlister, log, ml.WishlisterNames); err != nil {
		return err
	}
	if ml.guessers, err = getModulesAs[polochon.Guesser](mp, polochon.TypeGuesser, log, ml.GuesserNames); err != nil {
		return err
	}

	if ml.calendar, err = getModuleAs[polochon.Calendar](mp, polochon.TypeCalendar, log, ml.CalendarName); err != nil {
		return err
	}
	if ml.fsNotifier, err = getModuleAs[polochon.FsNotifier](mp, polochon.TypeFsNotifier, log, ml.FsNotifierName); err != nil {
		return err
	}
	if ml.downloader, err = getModuleAs[polochon.Downloader](mp, polochon.TypeDownloader, log, ml.DownloaderName); err != nil {
		return err
	}

	return nil
}
